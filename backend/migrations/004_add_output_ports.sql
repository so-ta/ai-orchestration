-- Migration: Add output ports to blocks and source_port to edges
-- This enables multiple outputs from blocks (e.g., condition: true/false)

-- Add output_ports to block_definitions
ALTER TABLE block_definitions ADD COLUMN IF NOT EXISTS output_ports JSONB DEFAULT '[]';

-- Add source_port to edges (which output port this edge connects from)
ALTER TABLE edges ADD COLUMN IF NOT EXISTS source_port VARCHAR(50) DEFAULT '';

-- Update built-in blocks with their output ports

-- Start: single output
UPDATE block_definitions SET output_ports = '[
  {"name": "output", "label": "Output", "description": "Workflow input data", "is_default": true}
]'::jsonb WHERE slug = 'start';

-- LLM: single output
UPDATE block_definitions SET output_ports = '[
  {"name": "output", "label": "Output", "description": "LLM response", "is_default": true, "schema": {"type": "object", "properties": {"content": {"type": "string"}, "tokens_used": {"type": "number"}}}}
]'::jsonb WHERE slug = 'llm';

-- Router: dynamic routes (configured via cases in step config)
UPDATE block_definitions SET output_ports = '[
  {"name": "default", "label": "Default", "description": "Default route when no match", "is_default": true}
]'::jsonb WHERE slug = 'router';

-- Condition: true/false outputs
UPDATE block_definitions SET output_ports = '[
  {"name": "true", "label": "Yes", "description": "When condition is true", "is_default": true},
  {"name": "false", "label": "No", "description": "When condition is false", "is_default": false}
]'::jsonb WHERE slug = 'condition';

-- Switch: dynamic cases (cases are defined in step config, we provide default only)
UPDATE block_definitions SET output_ports = '[
  {"name": "default", "label": "Default", "description": "When no case matches", "is_default": true}
]'::jsonb WHERE slug = 'switch';

-- Loop: loop body and completion
UPDATE block_definitions SET output_ports = '[
  {"name": "loop", "label": "Loop Body", "description": "Each iteration", "is_default": true},
  {"name": "complete", "label": "Complete", "description": "When loop finishes", "is_default": false}
]'::jsonb WHERE slug = 'loop';

-- Map: item output and completion
UPDATE block_definitions SET output_ports = '[
  {"name": "item", "label": "Item", "description": "Each mapped item", "is_default": true},
  {"name": "complete", "label": "Complete", "description": "All items processed", "is_default": false}
]'::jsonb WHERE slug = 'map';

-- Filter: matched and unmatched
UPDATE block_definitions SET output_ports = '[
  {"name": "matched", "label": "Matched", "description": "Items matching condition", "is_default": true},
  {"name": "unmatched", "label": "Unmatched", "description": "Items not matching", "is_default": false}
]'::jsonb WHERE slug = 'filter';

-- Human in Loop: approved and rejected
UPDATE block_definitions SET output_ports = '[
  {"name": "approved", "label": "Approved", "description": "When approved", "is_default": true},
  {"name": "rejected", "label": "Rejected", "description": "When rejected", "is_default": false},
  {"name": "timeout", "label": "Timeout", "description": "When timed out", "is_default": false}
]'::jsonb WHERE slug = 'human_in_loop';

-- Single output blocks
UPDATE block_definitions SET output_ports = '[
  {"name": "output", "label": "Output", "description": "Tool execution result", "is_default": true}
]'::jsonb WHERE slug = 'tool';

UPDATE block_definitions SET output_ports = '[
  {"name": "output", "label": "Output", "description": "Function result", "is_default": true}
]'::jsonb WHERE slug = 'function';

UPDATE block_definitions SET output_ports = '[
  {"name": "output", "label": "Output", "description": "Subflow result", "is_default": true}
]'::jsonb WHERE slug = 'subflow';

UPDATE block_definitions SET output_ports = '[
  {"name": "output", "label": "Output", "description": "Merged data", "is_default": true}
]'::jsonb WHERE slug = 'join';

UPDATE block_definitions SET output_ports = '[
  {"name": "output", "label": "Output", "description": "Split batches", "is_default": true}
]'::jsonb WHERE slug = 'split';

UPDATE block_definitions SET output_ports = '[
  {"name": "output", "label": "Output", "description": "Aggregated result", "is_default": true}
]'::jsonb WHERE slug = 'aggregate';

UPDATE block_definitions SET output_ports = '[
  {"name": "output", "label": "Output", "description": "Continues after wait", "is_default": true}
]'::jsonb WHERE slug = 'wait';

-- Error and Note have no outputs (terminal nodes)
UPDATE block_definitions SET output_ports = '[]'::jsonb WHERE slug = 'error';
UPDATE block_definitions SET output_ports = '[]'::jsonb WHERE slug = 'note';

-- Add index for source_port queries
CREATE INDEX IF NOT EXISTS idx_edges_source_port ON edges(source_step_id, source_port);

-- Update existing edges to have empty source_port (will use default output)
UPDATE edges SET source_port = '' WHERE source_port IS NULL;
