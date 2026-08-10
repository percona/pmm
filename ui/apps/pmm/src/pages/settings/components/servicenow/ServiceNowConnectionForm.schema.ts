import { z } from 'zod';
import { Messages } from '../../Settings.messages';

const { invalidUrl } = Messages.serviceNow.validation;

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
 * the schema only catches an endpoint that could never be a URL. The secret
 * values are positional and unconstrained — the names they belong to come from
 * the declared plan, and a mismatch is SEP's 422 to report, which the form
 * surfaces verbatim.
 *
 * An empty endpoint is valid: it means "keep the receiver this image bakes in".
 * An empty secret is valid too, and saves as an explicitly unconfigured state.
 */
export const serviceNowSchema = z.object({
  endpoint: z
    .string()
    .refine((value) => value.trim() === '' || isAbsoluteUrl(value.trim()), {
      message: invalidUrl,
    }),
  secrets: z.array(z.string()),
});
