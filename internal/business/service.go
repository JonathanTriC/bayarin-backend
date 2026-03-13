package business

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

// Business represents the business entity.
type Business struct {
	ID                    uuid.UUID `json:"id"`
	Name                  string    `json:"name"`
	Slug                  string    `json:"slug"`
	TaxPercent            float64   `json:"tax_percent"`
	ServiceChargePercent  float64   `json:"service_charge_percent"`
	CreatedAt             string    `json:"created_at"`
}

// UpdateBusinessInput holds fields that can be updated.
type UpdateBusinessInput struct {
	Name                 *string  `json:"name"`
	TaxPercent           *float64 `json:"tax_percent"`
	ServiceChargePercent *float64 `json:"service_charge_percent"`
}

// Service handles business-level operations.
type Service struct {
	db *sql.DB
}

// NewService creates a new business service.
func NewService(db *sql.DB) *Service { return &Service{db: db} }

// Get returns the business for the given ID.
func (s *Service) Get(businessID uuid.UUID) (*Business, error) {
	var b Business
	row := s.db.QueryRow(
		`SELECT id, name, slug, tax_percent, service_charge_percent, created_at
		 FROM businesses WHERE id = $1`, businessID,
	)
	err := row.Scan(&b.ID, &b.Name, &b.Slug, &b.TaxPercent, &b.ServiceChargePercent, &b.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("business not found")
		}
		return nil, err
	}
	return &b, nil
}

// Update applies partial updates to the business.
func (s *Service) Update(businessID uuid.UUID, input UpdateBusinessInput) (*Business, error) {
	current, err := s.Get(businessID)
	if err != nil {
		return nil, err
	}

	name := current.Name
	taxPct := current.TaxPercent
	svcPct := current.ServiceChargePercent

	if input.Name != nil {
		name = *input.Name
	}
	if input.TaxPercent != nil {
		taxPct = *input.TaxPercent
	}
	if input.ServiceChargePercent != nil {
		svcPct = *input.ServiceChargePercent
	}

	_, err = s.db.Exec(
		`UPDATE businesses SET name=$1, tax_percent=$2, service_charge_percent=$3 WHERE id=$4`,
		name, taxPct, svcPct, businessID,
	)
	if err != nil {
		return nil, fmt.Errorf("update business: %w", err)
	}

	current.Name = name
	current.TaxPercent = taxPct
	current.ServiceChargePercent = svcPct
	return current, nil
}
