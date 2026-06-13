CREATE TABLE IF NOT EXISTS student_works (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    teacher_id UUID NOT NULL REFERENCES teachers(id) ON DELETE CASCADE,
    session_id UUID,
    student_id UUID,
    image_path VARCHAR(500) NOT NULL,
    subject VARCHAR(50) NOT NULL,
    task_text TEXT NOT NULL,
    expected_answer TEXT,
    recognized_text TEXT,
    analysis_result JSONB,
    grade SMALLINT,
    teacher_comment TEXT,
    status VARCHAR(30) NOT NULL DEFAULT 'uploaded'
        CHECK (status IN ('uploaded', 'recognized', 'analyzed', 'checked', 'failed')),
    checked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS work_errors (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_work_id UUID NOT NULL REFERENCES student_works(id) ON DELETE CASCADE,
    error_type VARCHAR(50) NOT NULL,
    fragment TEXT NOT NULL,
    explanation TEXT NOT NULL,
    severity VARCHAR(20) NOT NULL DEFAULT 'minor',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS student_work_revisions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_work_id UUID NOT NULL REFERENCES student_works(id) ON DELETE CASCADE,
    recognized_text TEXT NOT NULL,
    snapshot_json JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Индексы для ускорения запросов по teacher_id и статусу
CREATE INDEX idx_student_works_teacher_id ON student_works(teacher_id);
CREATE INDEX idx_student_works_status ON student_works(status);
CREATE INDEX idx_work_errors_student_work_id ON work_errors(student_work_id);
CREATE INDEX idx_student_work_revisions_student_work_id ON student_work_revisions(student_work_id);