-- Add log block definition
-- The log block outputs messages for debugging and can be viewed in Run details

INSERT INTO block_definitions (tenant_id, slug, name, description, category, icon, executor_type, config_schema, error_codes) VALUES
(NULL, 'log', 'Log', 'Output log messages for debugging', 'utility', 'terminal', 'builtin',
    '{"type":"object","properties":{"message":{"type":"string","description":"Log message (supports {{$.field}} template variables)"},"level":{"type":"string","enum":["debug","info","warn","error"],"default":"info","description":"Log level"},"data":{"type":"string","description":"JSON path to include additional data (e.g. $.input)"}}}',
    '[]'
);
