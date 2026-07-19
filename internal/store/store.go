// Package store is the PostgreSQL data-access layer (pgx/v5).
package store

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	pool *pgxpool.Pool
}

func New(ctx context.Context, databaseURL string) (*Store, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, err
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return &Store{pool: pool}, nil
}

func (s *Store) Pool() *pgxpool.Pool { return s.pool }
func (s *Store) Close()              { s.pool.Close() }

// Audit appends an immutable audit entry. Errors are returned so callers can log
// them, but audit failures must never mask the primary operation's result.
func (s *Store) Audit(ctx context.Context, orgID uuid.UUID, actorID *uuid.UUID, action, entity, entityID string, meta map[string]any) error {
	if meta == nil {
		meta = map[string]any{}
	}
	b, _ := json.Marshal(meta)
	_, err := s.pool.Exec(ctx,
		`INSERT INTO audit_log(org_id, actor_id, action, entity, entity_id, meta)
		 VALUES ($1,$2,$3,$4,$5,$6)`,
		orgID, actorID, action, entity, entityID, string(b))
	return err
}

// AuditList returns audit entries for an org, newest first.
func (s *Store) AuditList(ctx context.Context, orgID uuid.UUID, limit int) ([]map[string]any, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := s.pool.Query(ctx,
		`SELECT id, actor_id, action, entity, entity_id, meta, created_at
		 FROM audit_log WHERE org_id=$1 ORDER BY created_at DESC LIMIT $2`, orgID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id uuid.UUID
		var actor *uuid.UUID
		var action, entity, entityID string
		var meta map[string]any
		var createdAt any
		if err := rows.Scan(&id, &actor, &action, &entity, &entityID, &meta, &createdAt); err != nil {
			return nil, err
		}
		out = append(out, map[string]any{
			"id": id, "actor_id": actor, "action": action,
			"entity": entity, "entity_id": entityID, "meta": meta, "created_at": createdAt,
		})
	}
	return out, rows.Err()
}
