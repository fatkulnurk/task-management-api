CREATE TABLE idempotency_keys (
    user_id CHAR(36) NOT NULL,
    idempotency_key CHAR(36) NOT NULL,
    response_status SMALLINT NOT NULL,
    response_body LONGTEXT NOT NULL,
    expires_at TIMESTAMP(6) NOT NULL,
    created_at TIMESTAMP(6) NOT NULL,
    PRIMARY KEY(user_id, idempotency_key),
    INDEX(expires_at),
    FOREIGN KEY(user_id) REFERENCES users(id)
) ENGINE=InnoDB;
