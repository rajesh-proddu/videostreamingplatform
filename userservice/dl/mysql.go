package dl

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/go-sql-driver/mysql"

	"github.com/yourusername/videostreamingplatform/userservice/models"
)

// mysqlDuplicateEntry is the MySQL error number for a unique-constraint violation.
const mysqlDuplicateEntry = 1062

// MySQLStore is a MySQL-backed Store.
type MySQLStore struct {
	db *sql.DB
}

// NewMySQLStore creates a MySQL-backed Store.
func NewMySQLStore(db *sql.DB) *MySQLStore {
	return &MySQLStore{db: db}
}

// IsDuplicateKey reports whether err is a MySQL unique-constraint violation.
func IsDuplicateKey(err error) bool {
	var me *mysql.MySQLError
	return errors.As(err, &me) && me.Number == mysqlDuplicateEntry
}

// --- Users ---

func (s *MySQLStore) CreateUser(ctx context.Context, u *models.User) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO users (id, email, password_hash) VALUES (?, ?, ?)`,
		u.ID, u.Email, u.PasswordHash)
	return err
}

func (s *MySQLStore) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, email, password_hash, created_at, updated_at FROM users WHERE email = ?`, email)
	return scanUser(row)
}

func (s *MySQLStore) GetUserByID(ctx context.Context, id string) (*models.User, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, email, password_hash, created_at, updated_at FROM users WHERE id = ?`, id)
	return scanUser(row)
}

func scanUser(row *sql.Row) (*models.User, error) {
	u := &models.User{}
	err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	return u, nil
}

// --- Plans ---

func (s *MySQLStore) GetPlanByName(ctx context.Context, name string) (*models.Plan, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, name, amount_minor, currency, period_days, created_at FROM plans WHERE name = ?`, name)
	return scanPlan(row)
}

func (s *MySQLStore) GetPlanByID(ctx context.Context, id string) (*models.Plan, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, name, amount_minor, currency, period_days, created_at FROM plans WHERE id = ?`, id)
	return scanPlan(row)
}

func scanPlan(row *sql.Row) (*models.Plan, error) {
	p := &models.Plan{}
	err := row.Scan(&p.ID, &p.Name, &p.AmountMinor, &p.Currency, &p.PeriodDays, &p.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrPlanNotFound
	}
	if err != nil {
		return nil, err
	}
	return p, nil
}

// --- Subscriptions ---

func (s *MySQLStore) CreateSubscription(ctx context.Context, sub *models.Subscription) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO subscriptions (id, user_id, plan_id, status, current_period_end) VALUES (?, ?, ?, ?, ?)`,
		sub.ID, sub.UserID, sub.PlanID, sub.Status, sub.CurrentPeriodEnd)
	return err
}

func (s *MySQLStore) GetSubscriptionByID(ctx context.Context, id string) (*models.Subscription, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, user_id, plan_id, status, current_period_end, created_at, updated_at
		 FROM subscriptions WHERE id = ?`, id)
	return scanSubscription(row)
}

func (s *MySQLStore) UpdateSubscription(ctx context.Context, sub *models.Subscription) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE subscriptions SET status = ?, current_period_end = ? WHERE id = ?`,
		sub.Status, sub.CurrentPeriodEnd, sub.ID)
	return err
}

func (s *MySQLStore) GetActiveSubscription(ctx context.Context, userID string) (*models.Subscription, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, user_id, plan_id, status, current_period_end, created_at, updated_at
		 FROM subscriptions
		 WHERE user_id = ? AND status = 'ACTIVE' AND current_period_end > NOW()
		 ORDER BY current_period_end DESC LIMIT 1`, userID)
	return scanSubscription(row)
}

func (s *MySQLStore) GetOpenSubscription(ctx context.Context, userID, planID string) (*models.Subscription, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, user_id, plan_id, status, current_period_end, created_at, updated_at
		 FROM subscriptions
		 WHERE user_id = ? AND plan_id = ? AND status IN ('PENDING_PAYMENT','ACTIVE')
		 ORDER BY created_at DESC LIMIT 1`, userID, planID)
	return scanSubscription(row)
}

func (s *MySQLStore) ListStalePendingSubscriptions(ctx context.Context, cutoff time.Time) ([]*models.Subscription, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, user_id, plan_id, status, current_period_end, created_at, updated_at
		 FROM subscriptions WHERE status = 'PENDING_PAYMENT' AND created_at < ?`, cutoff)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var subs []*models.Subscription
	for rows.Next() {
		sub, err := scanSubscriptionRows(rows)
		if err != nil {
			return nil, err
		}
		subs = append(subs, sub)
	}
	return subs, rows.Err()
}

func scanSubscription(row *sql.Row) (*models.Subscription, error) {
	sub := &models.Subscription{}
	var end sql.NullTime
	err := row.Scan(&sub.ID, &sub.UserID, &sub.PlanID, &sub.Status, &end, &sub.CreatedAt, &sub.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrSubscriptionNotFound
	}
	if err != nil {
		return nil, err
	}
	if end.Valid {
		sub.CurrentPeriodEnd = &end.Time
	}
	return sub, nil
}

func scanSubscriptionRows(rows *sql.Rows) (*models.Subscription, error) {
	sub := &models.Subscription{}
	var end sql.NullTime
	if err := rows.Scan(&sub.ID, &sub.UserID, &sub.PlanID, &sub.Status, &end, &sub.CreatedAt, &sub.UpdatedAt); err != nil {
		return nil, err
	}
	if end.Valid {
		sub.CurrentPeriodEnd = &end.Time
	}
	return sub, nil
}

// --- Payments ---

func (s *MySQLStore) CreatePayment(ctx context.Context, p *models.Payment) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO payments (id, user_id, subscription_id, amount_minor, currency, status,
			provider, provider_order_id, provider_payment_id, payment_url, idempotency_key)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		p.ID, p.UserID, p.SubscriptionID, p.AmountMinor, p.Currency, p.Status,
		p.Provider, nullStr(p.ProviderOrderID), nullStr(p.ProviderPaymentID), nullStr(p.PaymentURL), p.IdempotencyKey)
	return err
}

func (s *MySQLStore) GetPaymentBySubscriptionID(ctx context.Context, subID string) (*models.Payment, error) {
	row := s.db.QueryRowContext(ctx, paymentSelect+` WHERE subscription_id = ? ORDER BY created_at DESC LIMIT 1`, subID)
	return scanPayment(row)
}

func (s *MySQLStore) GetPaymentByID(ctx context.Context, id string) (*models.Payment, error) {
	row := s.db.QueryRowContext(ctx, paymentSelect+` WHERE id = ?`, id)
	return scanPayment(row)
}

func (s *MySQLStore) GetPaymentByIdempotencyKey(ctx context.Context, key string) (*models.Payment, error) {
	row := s.db.QueryRowContext(ctx, paymentSelect+` WHERE idempotency_key = ?`, key)
	return scanPayment(row)
}

func (s *MySQLStore) UpdatePayment(ctx context.Context, p *models.Payment) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE payments SET status = ?, provider_order_id = ?, provider_payment_id = ? WHERE id = ?`,
		p.Status, nullStr(p.ProviderOrderID), nullStr(p.ProviderPaymentID), p.ID)
	return err
}

func (s *MySQLStore) ListPendingPayments(ctx context.Context) ([]*models.Payment, error) {
	rows, err := s.db.QueryContext(ctx, paymentSelect+` WHERE status IN ('CREATED','PENDING')`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var payments []*models.Payment
	for rows.Next() {
		p, err := scanPaymentRows(rows)
		if err != nil {
			return nil, err
		}
		payments = append(payments, p)
	}
	return payments, rows.Err()
}

const paymentSelect = `SELECT id, user_id, subscription_id, amount_minor, currency, status,
	provider, provider_order_id, provider_payment_id, payment_url, idempotency_key, created_at, updated_at
	FROM payments`

func scanPayment(row *sql.Row) (*models.Payment, error) {
	p := &models.Payment{}
	var orderID, paymentID, paymentURL sql.NullString
	err := row.Scan(&p.ID, &p.UserID, &p.SubscriptionID, &p.AmountMinor, &p.Currency, &p.Status,
		&p.Provider, &orderID, &paymentID, &paymentURL, &p.IdempotencyKey, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrPaymentNotFound
	}
	if err != nil {
		return nil, err
	}
	p.ProviderOrderID = orderID.String
	p.ProviderPaymentID = paymentID.String
	p.PaymentURL = paymentURL.String
	return p, nil
}

func scanPaymentRows(rows *sql.Rows) (*models.Payment, error) {
	p := &models.Payment{}
	var orderID, paymentID, paymentURL sql.NullString
	if err := rows.Scan(&p.ID, &p.UserID, &p.SubscriptionID, &p.AmountMinor, &p.Currency, &p.Status,
		&p.Provider, &orderID, &paymentID, &paymentURL, &p.IdempotencyKey, &p.CreatedAt, &p.UpdatedAt); err != nil {
		return nil, err
	}
	p.ProviderOrderID = orderID.String
	p.ProviderPaymentID = paymentID.String
	p.PaymentURL = paymentURL.String
	return p, nil
}

// --- Webhook dedupe ---

func (s *MySQLStore) IsWebhookEventProcessed(ctx context.Context, eventID string) (bool, error) {
	var one int
	err := s.db.QueryRowContext(ctx,
		`SELECT 1 FROM webhook_events WHERE provider_event_id = ?`, eventID).Scan(&one)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (s *MySQLStore) MarkWebhookEventProcessed(ctx context.Context, eventID string) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO webhook_events (provider_event_id) VALUES (?)`, eventID)
	if IsDuplicateKey(err) {
		return nil // already recorded
	}
	return err
}

func nullStr(v string) sql.NullString {
	if v == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: v, Valid: true}
}
