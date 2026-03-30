package inventory

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/rubendubeux/inventory-manager/internal/character"
	"github.com/rubendubeux/inventory-manager/models"
)

type CoinService struct {
	coinRepo *CoinRepository
	charRepo *character.Repository
}

func NewCoinService(coinRepo *CoinRepository, charRepo *character.Repository) *CoinService {
	return &CoinService{coinRepo: coinRepo, charRepo: charRepo}
}

func (s *CoinService) GetCoinPurse(ctx context.Context, characterID, requesterID uuid.UUID, requesterRole string) ([]models.CoinPurse, error) {
	if err := s.checkAccess(ctx, characterID, requesterID, requesterRole); err != nil {
		return nil, err
	}
	return s.coinRepo.GetCoinPurse(ctx, characterID)
}

func (s *CoinService) SetCoinBalance(ctx context.Context, characterID, coinTypeID, requesterID uuid.UUID, requesterRole string, amount float64) ([]models.CoinPurse, error) {
	if err := s.checkAccess(ctx, characterID, requesterID, requesterRole); err != nil {
		return nil, err
	}
	if amount < 0 {
		return nil, fmt.Errorf("amount cannot be negative")
	}
	if err := s.coinRepo.SetCoinBalance(ctx, characterID, coinTypeID, amount); err != nil {
		return nil, err
	}
	return s.coinRepo.GetCoinPurse(ctx, characterID)
}

type ConvertResult struct {
	From models.CoinPurse `json:"from"`
	To   models.CoinPurse `json:"to"`
}

// findEffectiveRate faz BFS no grafo de conversões para encontrar a taxa composta entre dois coins.
func findEffectiveRate(edges []ConversionEdge, fromID, toID uuid.UUID) (float64, bool) {
	if fromID == toID {
		return 1.0, true
	}
	type state struct {
		id   uuid.UUID
		rate float64
	}
	graph := make(map[uuid.UUID][]struct {
		to   uuid.UUID
		rate float64
	})
	for _, e := range edges {
		graph[e.FromID] = append(graph[e.FromID], struct {
			to   uuid.UUID
			rate float64
		}{e.ToID, e.Rate})
	}
	visited := make(map[uuid.UUID]bool)
	queue := []state{{fromID, 1.0}}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if visited[cur.id] {
			continue
		}
		visited[cur.id] = true
		if cur.id == toID {
			return cur.rate, true
		}
		for _, e := range graph[cur.id] {
			if !visited[e.to] {
				queue = append(queue, state{e.to, cur.rate * e.rate})
			}
		}
	}
	return 0, false
}

func (s *CoinService) ConvertCoins(ctx context.Context, characterID, fromCoinID, toCoinID, requesterID uuid.UUID, requesterRole string, amount float64) (*ConvertResult, error) {
	if err := s.checkAccess(ctx, characterID, requesterID, requesterRole); err != nil {
		return nil, err
	}
	if amount <= 0 {
		return nil, fmt.Errorf("amount must be greater than zero")
	}

	edges, err := s.coinRepo.ListAllConversionsForCharacter(ctx, characterID)
	if err != nil {
		return nil, err
	}
	rate, ok := findEffectiveRate(edges, fromCoinID, toCoinID)
	if !ok {
		return nil, ErrNoConversion
	}

	balance, err := s.coinRepo.GetCoinBalance(ctx, characterID, fromCoinID)
	if err != nil {
		return nil, err
	}
	if balance < amount {
		return nil, ErrInsufficientFunds
	}

	received := amount * rate

	if err := s.coinRepo.AddToBalance(ctx, characterID, fromCoinID, -amount); err != nil {
		return nil, fmt.Errorf("deduct from: %w", err)
	}
	if err := s.coinRepo.AddToBalance(ctx, characterID, toCoinID, received); err != nil {
		return nil, fmt.Errorf("add to: %w", err)
	}

	purse, err := s.coinRepo.GetCoinPurse(ctx, characterID)
	if err != nil {
		return nil, err
	}

	result := &ConvertResult{}
	for _, cp := range purse {
		if cp.CoinTypeID == fromCoinID {
			result.From = cp
		}
		if cp.CoinTypeID == toCoinID {
			result.To = cp
		}
	}
	return result, nil
}

func (s *CoinService) checkAccess(ctx context.Context, characterID, requesterID uuid.UUID, requesterRole string) error {
	if requesterRole == "gm" {
		return nil
	}
	ch, err := s.charRepo.GetCharacterByID(ctx, characterID)
	if err != nil {
		return err
	}
	if ch.OwnerUserID != requesterID {
		return ErrForbidden
	}
	return nil
}
