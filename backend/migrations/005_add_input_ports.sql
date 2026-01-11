-- Migration: Add input ports to block definitions and target_port to edges
-- This allows blocks to have multiple typed input connections

-- Add input_ports column to block_definitions
ALTER TABLE block_definitions ADD COLUMN IF NOT EXISTS input_ports JSONB DEFAULT '[]';

-- Add target_port column to edges
ALTER TABLE edges ADD COLUMN IF NOT EXISTS target_port VARCHAR(50) DEFAULT '';

-- Update built-in blocks with input ports

-- Start block: no inputs (entry point)
UPDATE block_definitions SET input_ports = '[]'::jsonb WHERE slug = 'start';

-- LLM block: single input (prompt context)
UPDATE block_definitions SET input_ports = '[
  {"name": "input", "label": "Input", "description": "Data available for prompt template", "required": false, "schema": {"type": "any"}}
]'::jsonb WHERE slug = 'llm';

-- Tool block: single input
UPDATE block_definitions SET input_ports = '[
  {"name": "input", "label": "Input", "description": "Input data for the tool", "required": false, "schema": {"type": "any"}}
]'::jsonb WHERE slug = 'tool';

-- Condition block: single input (the value to evaluate)
UPDATE block_definitions SET input_ports = '[
  {"name": "input", "label": "Input", "description": "Data to evaluate condition against", "required": true, "schema": {"type": "any"}}
]'::jsonb WHERE slug = 'condition';

-- Switch block: single input (value to switch on)
UPDATE block_definitions SET input_ports = '[
  {"name": "input", "label": "Input", "description": "Value to switch on", "required": true, "schema": {"type": "any"}}
]'::jsonb WHERE slug = 'switch';

-- Map block: array input
UPDATE block_definitions SET input_ports = '[
  {"name": "items", "label": "Items", "description": "Array of items to process", "required": true, "schema": {"type": "array", "items": {"type": "any"}}}
]'::jsonb WHERE slug = 'map';

-- Join block: multiple inputs (collects results from parallel branches)
UPDATE block_definitions SET input_ports = '[
  {"name": "input_1", "label": "Input 1", "description": "First branch result", "required": false, "schema": {"type": "any"}},
  {"name": "input_2", "label": "Input 2", "description": "Second branch result", "required": false, "schema": {"type": "any"}},
  {"name": "input_3", "label": "Input 3", "description": "Third branch result", "required": false, "schema": {"type": "any"}},
  {"name": "input_4", "label": "Input 4", "description": "Fourth branch result", "required": false, "schema": {"type": "any"}}
]'::jsonb WHERE slug = 'join';

-- Subflow block: single input
UPDATE block_definitions SET input_ports = '[
  {"name": "input", "label": "Input", "description": "Input data for subflow", "required": false, "schema": {"type": "any"}}
]'::jsonb WHERE slug = 'subflow';

-- Loop block: single input (initial value or array)
UPDATE block_definitions SET input_ports = '[
  {"name": "input", "label": "Input", "description": "Initial value or array to iterate", "required": true, "schema": {"type": "any"}}
]'::jsonb WHERE slug = 'loop';

-- Wait block: single input (pass-through)
UPDATE block_definitions SET input_ports = '[
  {"name": "input", "label": "Input", "description": "Data to pass through after wait", "required": false, "schema": {"type": "any"}}
]'::jsonb WHERE slug = 'wait';

-- Function block: single input
UPDATE block_definitions SET input_ports = '[
  {"name": "input", "label": "Input", "description": "Input data for function", "required": false, "schema": {"type": "any"}}
]'::jsonb WHERE slug = 'function';

-- Router block: single input (message to route)
UPDATE block_definitions SET input_ports = '[
  {"name": "input", "label": "Input", "description": "Message to analyze for routing", "required": true, "schema": {"type": "string"}}
]'::jsonb WHERE slug = 'router';

-- Human in Loop block: single input (context for approval)
UPDATE block_definitions SET input_ports = '[
  {"name": "input", "label": "Input", "description": "Context data for human review", "required": true, "schema": {"type": "any"}}
]'::jsonb WHERE slug = 'human_in_loop';

-- Filter block: array input
UPDATE block_definitions SET input_ports = '[
  {"name": "items", "label": "Items", "description": "Array of items to filter", "required": true, "schema": {"type": "array", "items": {"type": "any"}}}
]'::jsonb WHERE slug = 'filter';

-- Split block: single input (data to split)
UPDATE block_definitions SET input_ports = '[
  {"name": "input", "label": "Input", "description": "Data to split into branches", "required": true, "schema": {"type": "any"}}
]'::jsonb WHERE slug = 'split';

-- Aggregate block: multiple inputs (combine data from multiple sources)
UPDATE block_definitions SET input_ports = '[
  {"name": "input_1", "label": "Input 1", "description": "First data source", "required": false, "schema": {"type": "any"}},
  {"name": "input_2", "label": "Input 2", "description": "Second data source", "required": false, "schema": {"type": "any"}},
  {"name": "input_3", "label": "Input 3", "description": "Third data source", "required": false, "schema": {"type": "any"}},
  {"name": "input_4", "label": "Input 4", "description": "Fourth data source", "required": false, "schema": {"type": "any"}}
]'::jsonb WHERE slug = 'aggregate';

-- Error block: single input (error info)
UPDATE block_definitions SET input_ports = '[
  {"name": "error", "label": "Error", "description": "Error information to handle", "required": true, "schema": {"type": "object", "properties": {"code": {"type": "string"}, "message": {"type": "string"}}}}
]'::jsonb WHERE slug = 'error';

-- Note block: no inputs (documentation only)
UPDATE block_definitions SET input_ports = '[]'::jsonb WHERE slug = 'note';

-- Create index for faster queries
CREATE INDEX IF NOT EXISTS idx_edges_target_port ON edges(target_port) WHERE target_port != '';
