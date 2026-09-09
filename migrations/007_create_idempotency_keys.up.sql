CREATE TABLE idempotency_keys (
    user_id CHAR(36) NOT NULL,
    endpoint VARCHAR(100) NOT NULL,
    idempotency_key CHAR(36) NOT NULL,
    task_id CHAR(36) NOT NULL,
    request_hash CHAR(64) NOT NULL,
    response_status SMALLINT NOT NULL,
    response_body JSON NOT NULL,
    expires_at TIMESTAMP(6) NOT NULL,
    created_at TIMESTAMP(6) NOT NULL,
    updated_at TIMESTAMP(6) NOT NULL,
    PRIMARY KEY(user_id, endpoint, idempotency_key),
    INDEX(expires_at),
    FOREIGN KEY(user_id) REFERENCES users(id),
    FOREIGN KEY(task_id) REFERENCES tasks(id)
) ENGINE=InnoDB;
