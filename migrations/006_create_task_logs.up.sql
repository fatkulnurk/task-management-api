CREATE TABLE task_logs (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    task_id CHAR(36) NOT NULL,
    actor_id CHAR(36) NOT NULL,
    action VARCHAR(40) NOT NULL,
    changes JSON NOT NULL,
    created_at TIMESTAMP(6) NOT NULL,
    FOREIGN KEY(task_id) REFERENCES tasks(id),
    FOREIGN KEY(actor_id) REFERENCES users(id)
) ENGINE=InnoDB;
