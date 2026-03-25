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
  id_number_enc TEXT NOT NULL,
  email VARCHAR(128) NULL,
  resume_path VARCHAR(255) NULL,
  position_id BIGINT NULL,
  institution VARCHAR(128) NOT NULL,
  department VARCHAR(128) NOT NULL,
  team VARCHAR(128) NOT NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'new',
  created_by BIGINT NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_candidates_scope (institution, department, team),
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
  action VARCHAR(128) NOT NULL,
  module_name VARCHAR(64) NOT NULL,
  record_id VARCHAR(64) NOT NULL,
  details_json JSON NULL,
  ip_address VARCHAR(64) NULL,
  user_agent VARCHAR(255) NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_audit_module (module_name),
  INDEX idx_audit_action (action),
  INDEX idx_audit_created (created_at),
  CONSTRAINT fk_audit_user FOREIGN KEY (user_id) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

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
