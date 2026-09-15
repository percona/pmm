import { z } from 'zod';
import { Messages } from '../../Settings.messages';

const { invalidUrl, required } = Messages.serviceNow.validation;

const isAbsoluteUrl = (value: string) => {
  try {
    const { protocol } = new URL(value);
    return protocol === 'http:' || protocol === 'https:';
  } catch {
    return false;
  }
};

/**
 * Client-side validation is deliberately thin: SEP validates the whole object
 * on write and is the only authority on which secret names are acceptable, so
 * the schema checks presence and an endpoint that could never be a URL, and
 * leaves the rest to SEP's 422 — which the form surfaces verbatim.
 *
 * Every declared secret is required. The form is only ever reached to supply
 * credentials — nothing stored, stored values the plan no longer accepts, or a
 * deliberate renewal — so a blank field is an unfinished form rather than a
 * request to store nothing. Removing a connection is what Disconnect is for.
 *
 * An empty endpoint stays valid: it means "keep the receiver this image bakes
 * in".
 */
export const serviceNowSchema = z.object({
  endpoint: z
    .string()
    .refine((value) => value.trim() === '' || isAbsoluteUrl(value.trim()), {
      message: invalidUrl,
    }),
  secrets: z.array(z.string().min(1, { message: required })),
});
