-- name: InsertAuditLog :exec 
INSERT INTO audit_log(event_type, payload, received_at)
VALUES ($1, $2, NOW());
