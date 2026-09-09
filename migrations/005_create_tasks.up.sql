CREATE TABLE tasks (
    id CHAR(36) PRIMARY KEY,
    team_id CHAR(36) NOT NULL,
    creator_id CHAR(36) NOT NULL,
    assignee_id CHAR(36) NULL,
    title VARCHAR(255) NOT NULL,
    description LONGTEXT NOT NULL,
    status ENUM('todo','in_progress','done') NOT NULL,
    deleted_at TIMESTAMP(6) NULL,
    created_at TIMESTAMP(6) NOT NULL,
    updated_at TIMESTAMP(6) NOT NULL,
    INDEX task_scope(team_id, deleted_at, status, created_at),
    FOREIGN KEY(team_id) REFERENCES teams(id),
    FOREIGN KEY(creator_id) REFERENCES users(id),
    FOREIGN KEY(assignee_id) REFERENCES users(id)
) ENGINE=InnoDB;
