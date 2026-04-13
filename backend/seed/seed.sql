-- Seed user: test@example.com / password123
-- Bcrypt hash generated with cost 12
INSERT INTO users (id, name, email, password) VALUES (
    'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11',
    'Test User',
    'test@example.com',
    '$2a$12$4GTcHy3oSrLqioN.7smhuOcISzDDuMRL8WUrMtD7IAfQZZcFpKJra'
) ON CONFLICT (email) DO NOTHING;

-- Seed project
INSERT INTO projects (id, name, description, owner_id) VALUES (
    'b1eebc99-9c0b-4ef8-bb6d-6bb9bd380a22',
    'Website Redesign',
    'Q2 redesign of the company website',
    'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11'
) ON CONFLICT (id) DO NOTHING;

-- Seed tasks with different statuses
INSERT INTO tasks (id, title, description, status, priority, project_id, assignee_id, created_by, due_date) VALUES
(
    'c1eebc99-9c0b-4ef8-bb6d-6bb9bd380a31',
    'Design homepage mockup',
    'Create wireframes and high-fidelity mockups for the new homepage',
    'todo',
    'high',
    'b1eebc99-9c0b-4ef8-bb6d-6bb9bd380a22',
    'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11',
    'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11',
    '2026-04-30'
),
(
    'c2eebc99-9c0b-4ef8-bb6d-6bb9bd380a32',
    'Set up CI/CD pipeline',
    'Configure GitHub Actions for automated testing and deployment',
    'in_progress',
    'medium',
    'b1eebc99-9c0b-4ef8-bb6d-6bb9bd380a22',
    'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11',
    'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11',
    '2026-04-20'
),
(
    'c3eebc99-9c0b-4ef8-bb6d-6bb9bd380a33',
    'Write API documentation',
    'Document all REST endpoints with request/response examples',
    'done',
    'low',
    'b1eebc99-9c0b-4ef8-bb6d-6bb9bd380a22',
    NULL,
    'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11',
    NULL
) ON CONFLICT (id) DO NOTHING;
