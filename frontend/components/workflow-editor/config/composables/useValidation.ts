/**
 * JSON Schemaベースのバリデーションcomposable
 *
 * シンプルなバリデーションを提供。
 * 将来的にはajvを導入して完全なJSON Schema準拠バリデーションに拡張可能。
 */

import { ref, computed, type Ref } from 'vue';
import type {
  ConfigSchema,
  ConditionalSchema,
  JSONSchemaProperty,
  ValidationError,
  ValidationResult,
} from '../types/config-schema';

/**
 * エラーメッセージ生成関数の型
 */
export interface ValidationMessageContext {
  field: string;
  label: string;
  keyword: string;
  params?: Record<string, unknown>;
}

export type ValidationMessageFn = (ctx: ValidationMessageContext) => string;

/**
 * デフォルトのエラーメッセージ（日本語）
 */
const defaultMessages: Record<string, (ctx: ValidationMessageContext) => string> = {
  required: ({ label }) => `${label}は必須です`,
  minLength: ({ params }) => `${params?.min}文字以上で入力してください`,
  maxLength: ({ params }) => `${params?.max}文字以内で入力してください`,
  pattern: () => `形式が正しくありません`,
  'format.uri': () => `有効なURLを入力してください`,
  'format.email': () => `有効なメールアドレスを入力してください`,
  enum: () => `有効な値を選択してください`,
  minimum: ({ params }) => `${params?.min}以上の値を入力してください`,
  maximum: ({ params }) => `${params?.max}以下の値を入力してください`,
  integer: () => `整数を入力してください`,
  minItems: ({ params }) => `${params?.min}件以上必要です`,
  maxItems: ({ params }) => `${params?.max}件以内にしてください`,
};

/**
 * メッセージを生成するヘルパー
 */
function getMessage(
  keyword: string,
  ctx: Omit<ValidationMessageContext, 'keyword'>,
  customMessages?: Record<string, ValidationMessageFn>
): string {
  const fullCtx = { ...ctx, keyword };

  // Check custom messages first
  if (customMessages?.[keyword]) {
    return customMessages[keyword](fullCtx);
  }

  // Fall back to defaults
  const messageFn = defaultMessages[keyword];
  if (messageFn) {
    return messageFn(fullCtx);
  }

  return `Validation failed: ${keyword}`;
}

/**
 * 条件付き必須フィールドを評価
 */
function evaluateConditionalRequired(
  conditionalRules: ConditionalSchema[] | undefined,
  values: Record<string, unknown>
): Set<string> {
  const conditionallyRequired = new Set<string>();

  if (!conditionalRules) return conditionallyRequired;

  for (const rule of conditionalRules) {
    if (!rule.if?.properties) continue;

    // Check if condition matches
    let matches = true;
    for (const [field, constraint] of Object.entries(rule.if.properties)) {
      if ('const' in constraint && values[field] !== constraint.const) {
        matches = false;
        break;
      }
    }

    // Apply then/else
    const applicable = matches ? rule.then : rule.else;
    if (applicable?.required) {
      for (const field of applicable.required) {
        conditionallyRequired.add(field);
      }
    }
  }

  return conditionallyRequired;
}

/**
 * 単一フィールドのバリデーション
 */
function validateField(
  name: string,
  value: unknown,
  property: JSONSchemaProperty,
  required: boolean,
  customMessages?: Record<string, ValidationMessageFn>
): ValidationError[] {
  const errors: ValidationError[] = [];
  const label = property.title || name;
  const baseCtx = { field: name, label };

  // Required check
  if (required && (value === undefined || value === null || value === '')) {
    errors.push({
      field: name,
      message: getMessage('required', baseCtx, customMessages),
      keyword: 'required',
    });
    return errors; // Skip other validations if required fails
  }

  // Skip validation if value is empty and not required
  if (value === undefined || value === null || value === '') {
    return errors;
  }

  const { type, minimum, maximum, minLength, maxLength, pattern, format, enum: enumValues } = property;

  // Type-specific validation
  if (type === 'string' && typeof value === 'string') {
    if (minLength !== undefined && value.length < minLength) {
      errors.push({
        field: name,
        message: getMessage('minLength', { ...baseCtx, params: { min: minLength } }, customMessages),
        keyword: 'minLength',
      });
    }

    if (maxLength !== undefined && value.length > maxLength) {
      errors.push({
        field: name,
        message: getMessage('maxLength', { ...baseCtx, params: { max: maxLength } }, customMessages),
        keyword: 'maxLength',
      });
    }

    if (pattern) {
      const regex = new RegExp(pattern);
      if (!regex.test(value)) {
        errors.push({
          field: name,
          message: getMessage('pattern', baseCtx, customMessages),
          keyword: 'pattern',
        });
      }
    }

    if (format === 'uri') {
      try {
        new URL(value);
      } catch {
        errors.push({
          field: name,
          message: getMessage('format.uri', baseCtx, customMessages),
          keyword: 'format',
        });
      }
    }

    if (format === 'email') {
      const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
      if (!emailRegex.test(value)) {
        errors.push({
          field: name,
          message: getMessage('format.email', baseCtx, customMessages),
          keyword: 'format',
        });
      }
    }

    if (enumValues && !enumValues.includes(value)) {
      errors.push({
        field: name,
        message: getMessage('enum', baseCtx, customMessages),
        keyword: 'enum',
      });
    }
  }

  if ((type === 'number' || type === 'integer') && typeof value === 'number') {
    if (minimum !== undefined && value < minimum) {
      errors.push({
        field: name,
        message: getMessage('minimum', { ...baseCtx, params: { min: minimum } }, customMessages),
        keyword: 'minimum',
      });
    }

    if (maximum !== undefined && value > maximum) {
      errors.push({
        field: name,
        message: getMessage('maximum', { ...baseCtx, params: { max: maximum } }, customMessages),
        keyword: 'maximum',
      });
    }

    if (type === 'integer' && !Number.isInteger(value)) {
      errors.push({
        field: name,
        message: getMessage('integer', baseCtx, customMessages),
        keyword: 'type',
      });
    }
  }

  if (type === 'array' && Array.isArray(value)) {
    const { minItems, maxItems } = property;

    if (minItems !== undefined && value.length < minItems) {
      errors.push({
        field: name,
        message: getMessage('minItems', { ...baseCtx, params: { min: minItems } }, customMessages),
        keyword: 'minItems',
      });
    }

    if (maxItems !== undefined && value.length > maxItems) {
      errors.push({
        field: name,
        message: getMessage('maxItems', { ...baseCtx, params: { max: maxItems } }, customMessages),
        keyword: 'maxItems',
      });
    }
  }

  return errors;
}

/**
 * Validation options
 */
export interface ValidationOptions {
  messages?: Record<string, ValidationMessageFn>;
}

/**
 * スキーマ全体のバリデーション
 */
export function validateConfig(
  schema: ConfigSchema | null | undefined,
  values: Record<string, unknown>,
  options?: ValidationOptions
): ValidationResult {
  if (!schema || !schema.properties) {
    return { valid: true, errors: [] };
  }

  // Static required fields
  const staticRequired = new Set(schema.required || []);

  // Evaluate conditional required fields based on current values
  const conditionalRequired = evaluateConditionalRequired(schema.allOf, values);

  // Merge: a field is required if it's in static OR conditional
  const allRequired = new Set([...staticRequired, ...conditionalRequired]);

  const allErrors: ValidationError[] = [];

  for (const [name, property] of Object.entries(schema.properties)) {
    const value = values[name];
    const required = allRequired.has(name);
    const fieldErrors = validateField(name, value, property, required, options?.messages);
    allErrors.push(...fieldErrors);
  }

  return {
    valid: allErrors.length === 0,
    errors: allErrors,
  };
}

/**
 * useValidation composable
 *
 * @example
 * // Basic usage
 * const { isValid, getFieldError } = useValidation(schemaRef, valuesRef);
 *
 * @example
 * // With custom messages (i18n)
 * const { t } = useI18n();
 * const { isValid } = useValidation(schemaRef, valuesRef, {
 *   messages: {
 *     required: ({ label }) => t('validation.required', { field: label }),
 *     minLength: ({ params }) => t('validation.minLength', { min: params?.min }),
 *   }
 * });
 */
export function useValidation(
  schema: Ref<ConfigSchema | null | undefined>,
  values: Ref<Record<string, unknown>>,
  options?: ValidationOptions
) {
  const touched = ref<Set<string>>(new Set());

  const validationResult = computed(() => validateConfig(schema.value, values.value, options));

  const isValid = computed(() => validationResult.value.valid);

  const errors = computed(() => validationResult.value.errors);

  const errorsByField = computed(() => {
    const result: Record<string, ValidationError[]> = {};
    for (const error of errors.value) {
      if (!result[error.field]) {
        result[error.field] = [];
      }
      result[error.field].push(error);
    }
    return result;
  });

  // Only show errors for touched fields
  const visibleErrors = computed(() => {
    const result: Record<string, ValidationError[]> = {};
    for (const [field, fieldErrors] of Object.entries(errorsByField.value)) {
      if (touched.value.has(field)) {
        result[field] = fieldErrors;
      }
    }
    return result;
  });

  function touch(field: string) {
    touched.value.add(field);
  }

  function touchAll() {
    if (schema.value?.properties) {
      for (const name of Object.keys(schema.value.properties)) {
        touched.value.add(name);
      }
    }
  }

  function resetTouched() {
    touched.value.clear();
  }

  function getFieldError(field: string): string | undefined {
    const fieldErrors = visibleErrors.value[field];
    return fieldErrors?.[0]?.message;
  }

  function hasFieldError(field: string): boolean {
    return (visibleErrors.value[field]?.length || 0) > 0;
  }

  return {
    isValid,
    errors,
    errorsByField,
    visibleErrors,
    touched,
    touch,
    touchAll,
    resetTouched,
    getFieldError,
    hasFieldError,
    validate: () => validationResult.value,
  };
}
