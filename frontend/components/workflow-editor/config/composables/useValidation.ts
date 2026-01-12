/**
 * JSON Schemaベースのバリデーションcomposable
 *
 * シンプルなバリデーションを提供。
 * 将来的にはajvを導入して完全なJSON Schema準拠バリデーションに拡張可能。
 */

import { ref, computed, type Ref } from 'vue';
import type {
  ConfigSchema,
  JSONSchemaProperty,
  ValidationError,
  ValidationResult,
} from '../types/config-schema';

/**
 * 単一フィールドのバリデーション
 */
function validateField(
  name: string,
  value: unknown,
  property: JSONSchemaProperty,
  required: boolean
): ValidationError[] {
  const errors: ValidationError[] = [];

  // Required check
  if (required && (value === undefined || value === null || value === '')) {
    errors.push({
      field: name,
      message: `${property.title || name}は必須です`,
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
        message: `${minLength}文字以上で入力してください`,
        keyword: 'minLength',
      });
    }

    if (maxLength !== undefined && value.length > maxLength) {
      errors.push({
        field: name,
        message: `${maxLength}文字以内で入力してください`,
        keyword: 'maxLength',
      });
    }

    if (pattern) {
      const regex = new RegExp(pattern);
      if (!regex.test(value)) {
        errors.push({
          field: name,
          message: `形式が正しくありません`,
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
          message: `有効なURLを入力してください`,
          keyword: 'format',
        });
      }
    }

    if (format === 'email') {
      const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
      if (!emailRegex.test(value)) {
        errors.push({
          field: name,
          message: `有効なメールアドレスを入力してください`,
          keyword: 'format',
        });
      }
    }

    if (enumValues && !enumValues.includes(value)) {
      errors.push({
        field: name,
        message: `有効な値を選択してください`,
        keyword: 'enum',
      });
    }
  }

  if ((type === 'number' || type === 'integer') && typeof value === 'number') {
    if (minimum !== undefined && value < minimum) {
      errors.push({
        field: name,
        message: `${minimum}以上の値を入力してください`,
        keyword: 'minimum',
      });
    }

    if (maximum !== undefined && value > maximum) {
      errors.push({
        field: name,
        message: `${maximum}以下の値を入力してください`,
        keyword: 'maximum',
      });
    }

    if (type === 'integer' && !Number.isInteger(value)) {
      errors.push({
        field: name,
        message: `整数を入力してください`,
        keyword: 'type',
      });
    }
  }

  if (type === 'array' && Array.isArray(value)) {
    const { minItems, maxItems } = property;

    if (minItems !== undefined && value.length < minItems) {
      errors.push({
        field: name,
        message: `${minItems}件以上必要です`,
        keyword: 'minItems',
      });
    }

    if (maxItems !== undefined && value.length > maxItems) {
      errors.push({
        field: name,
        message: `${maxItems}件以内にしてください`,
        keyword: 'maxItems',
      });
    }
  }

  return errors;
}

/**
 * スキーマ全体のバリデーション
 */
export function validateConfig(
  schema: ConfigSchema | null | undefined,
  values: Record<string, unknown>
): ValidationResult {
  if (!schema || !schema.properties) {
    return { valid: true, errors: [] };
  }

  const requiredFields = new Set(schema.required || []);
  const allErrors: ValidationError[] = [];

  for (const [name, property] of Object.entries(schema.properties)) {
    const value = values[name];
    const required = requiredFields.has(name);
    const fieldErrors = validateField(name, value, property, required);
    allErrors.push(...fieldErrors);
  }

  return {
    valid: allErrors.length === 0,
    errors: allErrors,
  };
}

/**
 * useValidation composable
 */
export function useValidation(
  schema: Ref<ConfigSchema | null | undefined>,
  values: Ref<Record<string, unknown>>
) {
  const touched = ref<Set<string>>(new Set());

  const validationResult = computed(() => validateConfig(schema.value, values.value));

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
