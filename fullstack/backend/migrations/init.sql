CREATE TABLE IF NOT EXISTS roles (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  code VARCHAR(64) NOT NULL UNIQUE,
  name VARCHAR(128) NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS data_scopes (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  institution VARCHAR(128) NOT NULL,
  department VARCHAR(128) NOT NULL,
  team VARCHAR(128) NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY uq_scope (institution, department, team)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS users (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  username VARCHAR(64) NOT NULL UNIQUE,
  password_hash VARCHAR(255) NOT NULL,
  full_name VARCHAR(128) NOT NULL,
  role_id BIGINT NOT NULL,
  data_scope_id BIGINT NOT NULL,
  is_active TINYINT(1) NOT NULL DEFAULT 1,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  CONSTRAINT fk_users_role FOREIGN KEY (role_id) REFERENCES roles(id),
  CONSTRAINT fk_users_scope FOREIGN KEY (data_scope_id) REFERENCES data_scopes(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS token_blacklist (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  jti VARCHAR(64) NOT NULL UNIQUE,
  user_id BIGINT NOT NULL,
  expires_at DATETIME NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_blacklist_user FOREIGN KEY (user_id) REFERENCES users(id),
  INDEX idx_blacklist_exp (expires_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS positions (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  title VARCHAR(128) NOT NULL,
  description TEXT NULL,
  required_skills_text TEXT NULL,
  required_education_level VARCHAR(64) NULL,
  min_years_experience DECIMAL(5,2) NULL,
  target_time_to_fill_days INT NOT NULL DEFAULT 30,
  tags_json JSON NULL,
  institution VARCHAR(128) NOT NULL,
  department VARCHAR(128) NOT NULL,
  team VARCHAR(128) NOT NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'open',
  created_by BIGINT NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_positions_scope (institution, department, team),
  CONSTRAINT fk_positions_creator FOREIGN KEY (created_by) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS candidates (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  full_name VARCHAR(128) NOT NULL,
  phone_enc TEXT NOT NULL,
  phone_hash CHAR(64) NULL,
  id_number_enc TEXT NOT NULL,
  id_number_hash CHAR(64) NULL,
  email VARCHAR(128) NULL,
  resume_path VARCHAR(255) NULL,
  tags_json JSON NULL,
  custom_fields_json JSON NULL,
  skills_text TEXT NULL,
  education_level VARCHAR(64) NULL,
  years_experience DECIMAL(5,2) NULL,
  last_active_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  position_id BIGINT NULL,
  institution VARCHAR(128) NOT NULL,
  department VARCHAR(128) NOT NULL,
  team VARCHAR(128) NOT NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'new',
  created_by BIGINT NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_candidates_scope (institution, department, team),
  INDEX idx_candidates_phone_hash (phone_hash),
  INDEX idx_candidates_id_hash (id_number_hash),
  CONSTRAINT fk_candidates_position FOREIGN KEY (position_id) REFERENCES positions(id),
  CONSTRAINT fk_candidates_creator FOREIGN KEY (created_by) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS qualifications (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  entity_type VARCHAR(32) NOT NULL,
  entity_name VARCHAR(128) NOT NULL,
  qualification_code VARCHAR(128) NOT NULL,
  issue_date DATE NOT NULL,
  expiry_date DATE NOT NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'active',
  notes_enc TEXT NULL,
  institution VARCHAR(128) NOT NULL,
  department VARCHAR(128) NOT NULL,
  team VARCHAR(128) NOT NULL,
  created_by BIGINT NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_qual_scope (institution, department, team),
  INDEX idx_qual_expiry (expiry_date),
  CONSTRAINT fk_qual_creator FOREIGN KEY (created_by) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS restrictions (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  med_name VARCHAR(128) NOT NULL,
  rule_type VARCHAR(64) NOT NULL,
  max_quantity DECIMAL(12,2) NOT NULL,
  requires_approval TINYINT(1) NOT NULL DEFAULT 1,
  requires_prescription TINYINT(1) NOT NULL DEFAULT 1,
  min_interval_days INT NOT NULL DEFAULT 7,
  fee_amount DECIMAL(12,2) NOT NULL DEFAULT 0.00,
  fee_currency VARCHAR(8) NOT NULL DEFAULT 'USD',
  institution VARCHAR(128) NOT NULL,
  department VARCHAR(128) NOT NULL,
  team VARCHAR(128) NOT NULL,
  is_active TINYINT(1) NOT NULL DEFAULT 1,
  created_by BIGINT NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_rest_scope (institution, department, team),
  UNIQUE KEY uq_restriction_rule (med_name, rule_type, institution, department, team),
  CONSTRAINT fk_rest_creator FOREIGN KEY (created_by) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS case_ledgers (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  case_no VARCHAR(64) NOT NULL UNIQUE,
  subject VARCHAR(200) NOT NULL,
  description_enc TEXT NOT NULL,
  institution VARCHAR(128) NOT NULL,
  department VARCHAR(128) NOT NULL,
  team VARCHAR(128) NOT NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'new',
  assigned_to BIGINT NULL,
  created_by BIGINT NOT NULL,
  last_transition_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_cases_scope (institution, department, team),
  INDEX idx_cases_subject_created (subject, created_at),
  CONSTRAINT fk_cases_creator FOREIGN KEY (created_by) REFERENCES users(id),
  CONSTRAINT fk_cases_assignee FOREIGN KEY (assigned_to) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS case_processing_records (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  case_id BIGINT NOT NULL,
  action_type VARCHAR(64) NOT NULL,
  from_status VARCHAR(32) NULL,
  to_status VARCHAR(32) NULL,
  note VARCHAR(255) NULL,
  assigned_to BIGINT NULL,
  details_json JSON NULL,
  institution VARCHAR(128) NOT NULL,
  department VARCHAR(128) NOT NULL,
  team VARCHAR(128) NOT NULL,
  changed_by BIGINT NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_case_processing_case (case_id, created_at),
  INDEX idx_case_processing_scope (institution, department, team),
  CONSTRAINT fk_case_processing_case FOREIGN KEY (case_id) REFERENCES case_ledgers(id),
  CONSTRAINT fk_case_processing_actor FOREIGN KEY (changed_by) REFERENCES users(id),
  CONSTRAINT fk_case_processing_assignee FOREIGN KEY (assigned_to) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS compliance_purchase_records (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  restriction_id BIGINT NOT NULL,
  client_id VARCHAR(64) NOT NULL,
  med_name VARCHAR(128) NOT NULL,
  quantity DECIMAL(12,2) NOT NULL,
  prescription_attachment_id BIGINT NOT NULL,
  institution VARCHAR(128) NOT NULL,
  department VARCHAR(128) NOT NULL,
  team VARCHAR(128) NOT NULL,
  reviewed_by BIGINT NOT NULL,
  details_json JSON NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_purchase_client_created (client_id, created_at),
  INDEX idx_purchase_scope (institution, department, team),
  CONSTRAINT fk_purchase_restriction FOREIGN KEY (restriction_id) REFERENCES restrictions(id),
  CONSTRAINT fk_purchase_user FOREIGN KEY (reviewed_by) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS attachments (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  module_name VARCHAR(64) NOT NULL,
  record_id BIGINT NOT NULL,
  original_name VARCHAR(255) NOT NULL,
  stored_name VARCHAR(255) NOT NULL,
  file_path VARCHAR(255) NOT NULL,
  mime_type VARCHAR(128) NOT NULL,
  file_size BIGINT NOT NULL,
  sha256 CHAR(64) NOT NULL,
  uploaded_by BIGINT NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY uq_attachments_sha (sha256),
  INDEX idx_attachments_module_record (module_name, record_id),
  CONSTRAINT fk_attachments_uploader FOREIGN KEY (uploaded_by) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS upload_sessions (
  id VARCHAR(64) PRIMARY KEY,
  module_name VARCHAR(64) NOT NULL,
  record_id BIGINT NOT NULL,
  original_name VARCHAR(255) NOT NULL,
  mime_type VARCHAR(128) NOT NULL,
  total_chunks INT NOT NULL,
  file_size BIGINT NOT NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'in_progress',
  uploaded_by BIGINT NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  CONSTRAINT fk_upload_sessions_uploader FOREIGN KEY (uploaded_by) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS audit_logs (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  user_id BIGINT NOT NULL,
  category VARCHAR(64) NOT NULL DEFAULT 'general',
  level VARCHAR(16) NOT NULL DEFAULT 'INFO',
  action VARCHAR(128) NOT NULL,
  module_name VARCHAR(64) NOT NULL,
  record_id VARCHAR(64) NOT NULL,
  before_json JSON NULL,
  after_json JSON NULL,
  diff_json JSON NULL,
  details_json JSON NULL,
  ip_address VARCHAR(64) NULL,
  user_agent VARCHAR(255) NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_audit_module (module_name),
  INDEX idx_audit_action (action),
  INDEX idx_audit_created (created_at),
  CONSTRAINT fk_audit_user FOREIGN KEY (user_id) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

ALTER TABLE positions ADD COLUMN IF NOT EXISTS required_skills_text TEXT NULL;
ALTER TABLE positions ADD COLUMN IF NOT EXISTS required_education_level VARCHAR(64) NULL;
ALTER TABLE positions ADD COLUMN IF NOT EXISTS min_years_experience DECIMAL(5,2) NULL;
ALTER TABLE positions ADD COLUMN IF NOT EXISTS target_time_to_fill_days INT NOT NULL DEFAULT 30;
ALTER TABLE positions ADD COLUMN IF NOT EXISTS tags_json JSON NULL;

ALTER TABLE candidates ADD COLUMN IF NOT EXISTS phone_hash CHAR(64) NULL;
ALTER TABLE candidates ADD COLUMN IF NOT EXISTS id_number_hash CHAR(64) NULL;
ALTER TABLE candidates ADD COLUMN IF NOT EXISTS tags_json JSON NULL;
ALTER TABLE candidates ADD COLUMN IF NOT EXISTS custom_fields_json JSON NULL;
ALTER TABLE candidates ADD COLUMN IF NOT EXISTS skills_text TEXT NULL;
ALTER TABLE candidates ADD COLUMN IF NOT EXISTS education_level VARCHAR(64) NULL;
ALTER TABLE candidates ADD COLUMN IF NOT EXISTS years_experience DECIMAL(5,2) NULL;
ALTER TABLE candidates ADD COLUMN IF NOT EXISTS last_active_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP;
ALTER TABLE restrictions ADD COLUMN IF NOT EXISTS requires_prescription TINYINT(1) NOT NULL DEFAULT 1;
ALTER TABLE restrictions ADD COLUMN IF NOT EXISTS min_interval_days INT NOT NULL DEFAULT 7;
ALTER TABLE restrictions ADD COLUMN IF NOT EXISTS fee_amount DECIMAL(12,2) NOT NULL DEFAULT 0.00;
ALTER TABLE restrictions ADD COLUMN IF NOT EXISTS fee_currency VARCHAR(8) NOT NULL DEFAULT 'USD';

ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS category VARCHAR(64) NOT NULL DEFAULT 'general';
ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS level VARCHAR(16) NOT NULL DEFAULT 'INFO';
ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS before_json JSON NULL;
ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS after_json JSON NULL;
ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS diff_json JSON NULL;

DROP TRIGGER IF EXISTS trg_audit_logs_block_update;
CREATE TRIGGER trg_audit_logs_block_update
BEFORE UPDATE ON audit_logs
FOR EACH ROW
SIGNAL SQLSTATE '45000'
SET MESSAGE_TEXT = 'audit_logs is append-only: UPDATE is forbidden';

DROP TRIGGER IF EXISTS trg_audit_logs_block_delete;
CREATE TRIGGER trg_audit_logs_block_delete
BEFORE DELETE ON audit_logs
FOR EACH ROW
SIGNAL SQLSTATE '45000'
SET MESSAGE_TEXT = 'audit_logs is append-only: DELETE is forbidden';

INSERT INTO roles (code, name)
VALUES
  ('business_specialist', 'Business Specialist'),
  ('compliance_admin', 'Compliance Admin'),
  ('recruitment_specialist', 'Recruitment Specialist'),
  ('system_admin', 'System Admin')
ON DUPLICATE KEY UPDATE name = VALUES(name);

INSERT INTO data_scopes (institution, department, team)
VALUES
  ('EAGLE_HOSPITAL', 'HQ', 'CORE'),
  ('EAGLE_HOSPITAL', 'COMPLIANCE', 'TEAM_A'),
  ('EAGLE_HOSPITAL', 'RECRUITMENT', 'TEAM_B'),
  ('EAGLE_HOSPITAL', 'RECRUITMENT', 'TEAM_D'),
  ('EAGLE_HOSPITAL', 'OPERATIONS', 'TEAM_C')
ON DUPLICATE KEY UPDATE institution = VALUES(institution);

INSERT INTO users (username, password_hash, full_name, role_id, data_scope_id, is_active)
SELECT
  'admin',
  '$2a$10$724DuJNRzNkhuYwOYpcRRua/oaFBTBcGm8mI7N2UEvTaGujkEP5ua',
  'System Administrator',
  r.id,
  ds.id,
  1
FROM roles r
JOIN data_scopes ds ON ds.institution = 'EAGLE_HOSPITAL' AND ds.department = 'HQ' AND ds.team = 'CORE'
WHERE r.code = 'system_admin'
ON DUPLICATE KEY UPDATE
  password_hash = VALUES(password_hash),
  role_id = VALUES(role_id),
  data_scope_id = VALUES(data_scope_id),
  is_active = VALUES(is_active);

INSERT INTO restrictions (med_name, rule_type, max_quantity, requires_approval, institution, department, team, is_active, created_by)
SELECT 'Morphine', 'controlled_purchase_limit', 50, 1, 'EAGLE_HOSPITAL', 'HQ', 'CORE', 1, u.id
FROM users u WHERE u.username = 'admin'
ON DUPLICATE KEY UPDATE max_quantity = VALUES(max_quantity), is_active = VALUES(is_active);

INSERT INTO users (username, password_hash, full_name, role_id, data_scope_id, is_active)
SELECT
  'recruiter_b',
  '$2a$10$724DuJNRzNkhuYwOYpcRRua/oaFBTBcGm8mI7N2UEvTaGujkEP5ua',
  'Recruiter Team B',
  r.id,
  ds.id,
  1
FROM roles r
JOIN data_scopes ds ON ds.institution = 'EAGLE_HOSPITAL' AND ds.department = 'RECRUITMENT' AND ds.team = 'TEAM_B'
WHERE r.code = 'recruitment_specialist'
ON DUPLICATE KEY UPDATE
  password_hash = VALUES(password_hash),
  role_id = VALUES(role_id),
  data_scope_id = VALUES(data_scope_id),
  is_active = VALUES(is_active);

INSERT INTO users (username, password_hash, full_name, role_id, data_scope_id, is_active)
SELECT
  'recruiter_d',
  '$2a$10$724DuJNRzNkhuYwOYpcRRua/oaFBTBcGm8mI7N2UEvTaGujkEP5ua',
  'Recruiter Team D',
  r.id,
  ds.id,
  1
FROM roles r
JOIN data_scopes ds ON ds.institution = 'EAGLE_HOSPITAL' AND ds.department = 'RECRUITMENT' AND ds.team = 'TEAM_D'
WHERE r.code = 'recruitment_specialist'
ON DUPLICATE KEY UPDATE
  password_hash = VALUES(password_hash),
  role_id = VALUES(role_id),
  data_scope_id = VALUES(data_scope_id),
  is_active = VALUES(is_active);

INSERT INTO users (username, password_hash, full_name, role_id, data_scope_id, is_active)
SELECT
  'compliance_a',
  '$2a$10$724DuJNRzNkhuYwOYpcRRua/oaFBTBcGm8mI7N2UEvTaGujkEP5ua',
  'Compliance Team A',
  r.id,
  ds.id,
  1
FROM roles r
JOIN data_scopes ds ON ds.institution = 'EAGLE_HOSPITAL' AND ds.department = 'COMPLIANCE' AND ds.team = 'TEAM_A'
WHERE r.code = 'compliance_admin'
ON DUPLICATE KEY UPDATE
  password_hash = VALUES(password_hash),
  role_id = VALUES(role_id),
  data_scope_id = VALUES(data_scope_id),
  is_active = VALUES(is_active);

INSERT INTO users (username, password_hash, full_name, role_id, data_scope_id, is_active)
SELECT
  'ops_case',
  '$2a$10$724DuJNRzNkhuYwOYpcRRua/oaFBTBcGm8mI7N2UEvTaGujkEP5ua',
  'Operations Case Specialist',
  r.id,
  ds.id,
  1
FROM roles r
JOIN data_scopes ds ON ds.institution = 'EAGLE_HOSPITAL' AND ds.department = 'OPERATIONS' AND ds.team = 'TEAM_C'
WHERE r.code = 'business_specialist'
ON DUPLICATE KEY UPDATE
  password_hash = VALUES(password_hash),
  role_id = VALUES(role_id),
  data_scope_id = VALUES(data_scope_id),
  is_active = VALUES(is_active);
