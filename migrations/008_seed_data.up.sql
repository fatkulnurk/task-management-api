INSERT INTO users(id,name,email,password_hash,created_at,updated_at) VALUES
('11111111-1111-4111-8111-111111111111','Alice Pratama','alice@fatkulnurk.com','$2a$10$u.yf2AnccCdFmOLZJVfheOAggKghj794ONNZrk2z1fTeEI046/CKu','2026-01-01 08:00:00.000000','2026-01-01 08:00:00.000000'),
('22222222-2222-4222-8222-222222222222','Bob Santoso','bob@fatkulnurk.com','$2a$10$u.yf2AnccCdFmOLZJVfheOAggKghj794ONNZrk2z1fTeEI046/CKu','2026-01-01 08:05:00.000000','2026-01-01 08:05:00.000000'),
('33333333-3333-4333-8333-333333333333','Carol Wijaya','carol@fatkulnurk.com','$2a$10$u.yf2AnccCdFmOLZJVfheOAggKghj794ONNZrk2z1fTeEI046/CKu','2026-01-01 08:10:00.000000','2026-01-01 08:10:00.000000');

INSERT INTO teams(id,owner_id,name,created_at,updated_at) VALUES
('aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa','11111111-1111-4111-8111-111111111111','Platform Team','2026-01-02 09:00:00.000000','2026-01-02 09:00:00.000000'),
('bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb','22222222-2222-4222-8222-222222222222','Mobile App Team','2026-01-02 09:30:00.000000','2026-01-02 09:30:00.000000');

INSERT INTO team_members(team_id,user_id) VALUES
('aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa','11111111-1111-4111-8111-111111111111'),
('aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa','22222222-2222-4222-8222-222222222222'),
('aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa','33333333-3333-4333-8333-333333333333'),
('bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb','22222222-2222-4222-8222-222222222222'),
('bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb','33333333-3333-4333-8333-333333333333');

INSERT INTO tasks(id,team_id,creator_id,assignee_id,title,description,status,deleted_at,created_at,updated_at) VALUES
('00000000-0000-4000-8000-000000000001','aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa','11111111-1111-4111-8111-111111111111','22222222-2222-4222-8222-222222222222','Set up CI pipeline','Configure build, test, and release automation for the platform services.','in_progress',NULL,'2026-01-03 10:00:00.000000','2026-01-05 14:30:00.000000'),
('00000000-0000-4000-8000-000000000002','aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa','11111111-1111-4111-8111-111111111111','11111111-1111-4111-8111-111111111111','Design database schema','Define tables, indexes, and foreign keys for the task management domain.','done',NULL,'2026-01-03 10:15:00.000000','2026-01-04 16:45:00.000000'),
('00000000-0000-4000-8000-000000000003','aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa','22222222-2222-4222-8222-222222222222','33333333-3333-4333-8333-333333333333','Write API documentation','Document every endpoint with request and response examples.','todo',NULL,'2026-01-03 10:30:00.000000','2026-01-03 10:30:00.000000'),
('00000000-0000-4000-8000-000000000004','aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa','11111111-1111-4111-8111-111111111111','11111111-1111-4111-8111-111111111111','Implement authentication','Add register, login, refresh, and logout flows backed by JWT.','in_progress',NULL,'2026-01-04 09:00:00.000000','2026-01-06 11:20:00.000000'),
('00000000-0000-4000-8000-000000000005','aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa','22222222-2222-4222-8222-222222222222',NULL,'Fix login rate limiting','Limit repeated failed login attempts per account and per IP.','todo',NULL,'2026-01-04 09:45:00.000000','2026-01-04 09:45:00.000000'),
('00000000-0000-4000-8000-000000000006','aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa','11111111-1111-4111-8111-111111111111',NULL,'Old onboarding flow','Superseded onboarding flow kept for historical reference.','todo','2026-01-07 08:00:00.000000','2026-01-05 07:30:00.000000','2026-01-07 08:00:00.000000'),
('00000000-0000-4000-8000-000000000007','bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb','22222222-2222-4222-8222-222222222222','33333333-3333-4333-8333-333333333333','Publish app store release','Prepare store listings and submit the release build for review.','todo',NULL,'2026-01-06 13:00:00.000000','2026-01-06 13:00:00.000000');

INSERT INTO task_logs(task_id,actor_id,action,changes,created_at) VALUES
('00000000-0000-4000-8000-000000000001','11111111-1111-4111-8111-111111111111','task_created',JSON_OBJECT('status','todo'),'2026-01-03 10:00:00.000000'),
('00000000-0000-4000-8000-000000000001','11111111-1111-4111-8111-111111111111','task_assigned',JSON_OBJECT('assignee_id','22222222-2222-4222-8222-222222222222'),'2026-01-03 10:20:00.000000'),
('00000000-0000-4000-8000-000000000001','22222222-2222-4222-8222-222222222222','task_status_changed',JSON_OBJECT('from','todo','to','in_progress'),'2026-01-05 14:30:00.000000'),
('00000000-0000-4000-8000-000000000002','11111111-1111-4111-8111-111111111111','task_created',JSON_OBJECT('status','todo'),'2026-01-03 10:15:00.000000'),
('00000000-0000-4000-8000-000000000002','11111111-1111-4111-8111-111111111111','task_status_changed',JSON_OBJECT('from','in_progress','to','done'),'2026-01-04 16:45:00.000000'),
('00000000-0000-4000-8000-000000000004','11111111-1111-4111-8111-111111111111','task_created',JSON_OBJECT('status','todo'),'2026-01-04 09:00:00.000000'),
('00000000-0000-4000-8000-000000000004','11111111-1111-4111-8111-111111111111','task_status_changed',JSON_OBJECT('from','todo','to','in_progress'),'2026-01-06 11:20:00.000000'),
('00000000-0000-4000-8000-000000000005','22222222-2222-4222-8222-222222222222','task_created',JSON_OBJECT('status','todo'),'2026-01-04 09:45:00.000000'),
('00000000-0000-4000-8000-000000000006','11111111-1111-4111-8111-111111111111','task_created',JSON_OBJECT('status','todo'),'2026-01-05 07:30:00.000000'),
('00000000-0000-4000-8000-000000000006','11111111-1111-4111-8111-111111111111','task_deleted',JSON_OBJECT('deleted_at','2026-01-07 08:00:00.000000'),'2026-01-07 08:00:00.000000'),
('00000000-0000-4000-8000-000000000007','22222222-2222-4222-8222-222222222222','task_created',JSON_OBJECT('status','todo'),'2026-01-06 13:00:00.000000');
