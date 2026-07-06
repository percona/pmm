// Copyright (C) 2023 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package logger

import (
	"strings"
	"sync"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"

	extensionsv1 "github.com/percona/pmm/api/extensions/v1"
)

// A messagePlan is the plan for redacting a message, which contains the information
// of which fields in this particular message are sensitive and how to redact them.
// NOTE: messagePlan only contains the information of the current-level message(md),
// not for nested messages(md) in fields, because we will also reflect on nested messages(md) when we redact them,
// and cache their plans as well.
type messagePlan struct {
	// A sensitive maps field number to its redact type, only for fields annotated with our custom "sensitive" option.
	// We use field number instead of field name for better performance,
	// as we can directly get the field value by number from protoreflect.Message.
	sensitive map[protoreflect.FieldNumber]extensionsv1.RedactType
}

var (
	// A messagePlan is cached by message descriptor full name, which is unique in protobuf world.
	// The cache value is *messagePlan if there are sensitive fields in this message(md),
	// or nil if there is no sensitive field in this message(md).
	// We use sync.Map for concurrent access, as protobuf messages can be redacted in multiple goroutines.
	// Note that we only cache the plan for the current-level message(md), not for nested messages(md) in fields,
	// because we will also reflect on nested messages(md) when we redact them, and cache their plans as well.
	planCache sync.Map // map[protoreflect.FullName]*messagePlan

	// A maskedString is used to replace sensitive string fields.
	maskedString = "***REDACTED***"
)

// RedactMessage returns a cloned message where annotated string fields are redacted.
func RedactMessage(msg proto.Message) proto.Message { //nolint:ireturn
	if msg == nil {
		return nil
	}

	cloned := proto.Clone(msg)
	redactMessageReflect(cloned.ProtoReflect())
	return cloned
}

func redactMessageReflect(m protoreflect.Message) {
	if !m.IsValid() {
		return
	}

	plan := getPlan(m.Descriptor())
	m.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		if plan != nil {
			if rt, ok := plan.sensitive[fd.Number()]; ok {
				applySensitive(m, fd, v, rt)
				return true
			}
		}

		switch {
		case fd.IsList():
			if fd.Kind() == protoreflect.MessageKind {
				list := v.List()
				for i := range list.Len() {
					redactMessageReflect(list.Get(i).Message())
				}
			}
		case fd.IsMap():
			if fd.MapValue().Kind() == protoreflect.MessageKind {
				mv := v.Map()
				mv.Range(func(_ protoreflect.MapKey, val protoreflect.Value) bool {
					redactMessageReflect(val.Message())
					return true
				})
			}
		case fd.Kind() == protoreflect.MessageKind:
			redactMessageReflect(v.Message())
		}

		return true
	})
}

func getPlan(md protoreflect.MessageDescriptor) *messagePlan {
	key := md.FullName()
	if p, ok := planCache.Load(key); ok {
		// Cache hit, can be nil if no sensitive fields in this message(md).
		if p == nil {
			return nil
		}

		if mp, ok := p.(*messagePlan); ok {
			return mp
		}
		return nil
	}

	// Cache miss, need to reflect on this message(md) to find sensitive fields.
	// Reflection is expensive, but we only do it once per message(md) and cache the result for future use.
	fields := md.Fields()
	sensitive := make(map[protoreflect.FieldNumber]extensionsv1.RedactType)
	for i := range fields.Len() {
		fd := fields.Get(i)
		opts, ok := fd.Options().(*descriptorpb.FieldOptions)
		if !ok || opts == nil {
			continue
		}

		ext := proto.GetExtension(opts, extensionsv1.E_Sensitive)
		rt, ok := ext.(extensionsv1.RedactType)
		if !ok || rt == extensionsv1.RedactType_REDACT_TYPE_UNSPECIFIED {
			continue
		}
		sensitive[fd.Number()] = rt
	}

	if len(sensitive) == 0 {
		// No sensitive fields, cache nil to avoid future reflection on this message(md).
		planCache.Store(key, nil)
		return nil
	}

	plan := &messagePlan{sensitive: sensitive}
	planCache.Store(key, plan)
	return plan
}

func applySensitive(m protoreflect.Message, fd protoreflect.FieldDescriptor, value protoreflect.Value, rt extensionsv1.RedactType) {
	switch {
	case fd.IsList():
		if fd.Kind() == protoreflect.StringKind {
			list := m.Get(fd).List()
			for i := range list.Len() {
				list.Set(i, protoreflect.ValueOfString(RedactString(list.Get(i).String(), rt)))
			}
		}
	case fd.IsMap():
		if fd.MapValue().Kind() == protoreflect.StringKind {
			mv := m.Get(fd).Map()
			mv.Range(func(k protoreflect.MapKey, v protoreflect.Value) bool {
				mv.Set(k, protoreflect.ValueOfString(RedactString(v.String(), rt)))
				return true
			})
		}
	case fd.Kind() == protoreflect.StringKind:
		m.Set(fd, protoreflect.ValueOfString(RedactString(value.String(), rt)))
	}
}

// RedactString returns a redacted copy of string based on the given redact type.
func RedactString(s string, rt extensionsv1.RedactType) string {
	switch rt {
	case extensionsv1.RedactType_REDACT_TYPE_FULL:
		return maskedString
	case extensionsv1.RedactType_REDACT_TYPE_MASK:
		return maskString(s)
	case extensionsv1.RedactType_REDACT_TYPE_DSN:
		return MaskDSN(s)
	default:
		return s
	}
}

func maskString(s string) string {
	n := len(s)
	switch {
	case n == 0:
		return s
	case n <= 4: //nolint:mnd
		return maskedString
	default:
		return s[:2] + maskedString + s[n-2:]
	}
}

// MaskDSN returns a masked copy of DSN string, which masks username and password in DSN.
func MaskDSN(s string) string {
	at := strings.LastIndex(s, "@")
	if at <= 0 {
		return s
	}

	left := s[:at]
	tail := s[at:]
	prefix := ""

	// Handle scheme://user:pass@host... by keeping scheme:// unchanged.
	if schemeSep := strings.Index(left, "://"); schemeSep >= 0 {
		credStart := schemeSep + 3 //nolint:mnd
		if credStart < len(left) {
			prefix = left[:credStart]
			left = left[credStart:]
		}
	}

	// user:password@host -> ****:****@host
	if colon := strings.LastIndexByte(left, ':'); colon >= 0 {
		return prefix + maskedString + ":" + maskedString + tail
	}
	return prefix + maskedString + tail
}
