CREATE TABLE IF NOT EXISTS hotspot_reports (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    report_date DATE NOT NULL,
    generated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    total_incidents INTEGER NOT NULL DEFAULT 0,
    hotspots JSONB NOT NULL DEFAULT '[]',
    temporal_insights JSONB NOT NULL DEFAULT '{}',
    category_distribution JSONB NOT NULL DEFAULT '{}',
    ai_summary TEXT,
    ai_recommendations JSONB NOT NULL DEFAULT '[]',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_hotspot_reports_report_date
    ON hotspot_reports (report_date);

CREATE INDEX IF NOT EXISTS idx_hotspot_reports_generated_at
    ON hotspot_reports (generated_at DESC);
