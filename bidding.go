package main

import (
	"errors"
	"github.com/google/uuid"
	"sync"
	"time"
)

// Bid is a bid on an item
type Bid struct {
	Email     string  `json:"email"`
	Amount    float64 `json:"amount"`
	Timestamp int64   `json:"timestamp"`
}

// NewBid creates a new bid on an item
func NewBid(email string, amount float64) *Bid {
	return &Bid{
		Email:     email,
		Amount:    amount,
		Timestamp: time.Now().Unix(),
	}
}

// Item is a product that is being auctioned
type Item struct {
	// ID is the unique identifier for the item
	ID uuid.UUID `json:"id"`

	// Name is the name of the item
	Name string `json:"name"`

	// ReservePrice is the minimum price that the seller is willing to accept
	ReservePrice float64 `json:"reservePrice"`

	// StartingAt is the time that the auction started
	StartingAt time.Time `json:"startingAt"`

	// EndAt is the time that the auction will end
	EndAt time.Time `json:"endAt"`
}

// NewItem creates a new item to be auctioned
func NewItem(name string, reservePrice float64, startingAt, endAt time.Time) *Item {
	return &Item{
		ID:           uuid.New(),
		Name:         name,
		ReservePrice: reservePrice,
		StartingAt:   startingAt,
		EndAt:        endAt,
	}
}

// ReservePriceMet returns true if the reserve price has been met
func (i *Item) ReservePriceMet(bidAmount float64) bool {
	return bidAmount >= i.ReservePrice
}

// AuctionOngoing returns true if the auction is ongoing
func (i *Item) AuctionOngoing() bool {
	return time.Now().UTC().Before(i.EndAt)
}

// Auction is an auction for an item
type Auction struct {
	// mu is a mutex to protect the bid
	mu sync.RWMutex

	// Item is the item that is being bid on
	Item *Item

	// CurrentWinner is the current highest bidder
	CurrentWinner *Bid `json:"currentWinner"`

	// Bids is a list of all bids that have been placed on the item
	Bids []Bid `json:"bids"`
}

func (a *Auction) ChangeTime(endAt time.Time) {
	a.Item.EndAt = endAt
}

// NewAuction creates a new auction for an item
func NewAuction(item *Item) *Auction {
	return &Auction{
		Item: item,
		mu:   sync.RWMutex{},
	}
}

var (
	ErrBidAmountBelowStartingBid = errors.New("bid amount is below starting bid")
	ErrBidAmountBelowCurrentBid  = errors.New("bid amount is below current bid")
	ErrAuctionOver               = errors.New("auction is over")
)

// TryBid attempts to place a bid on an item
func (a *Auction) TryBid(bid *Bid) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if err := a.validateBid(bid); err != nil {
		return err
	}

	a.setWinner(bid)
	a.addBidHistory(bid)

	return nil
}

func (a *Auction) validateBid(bid *Bid) error {
	if !a.Item.AuctionOngoing() {
		return ErrAuctionOver
	}

	if !a.Item.ReservePriceMet(bid.Amount) {
		return ErrBidAmountBelowStartingBid
	}

	// be careful of nil pointer dereference
	if a.CurrentWinner == nil {
		return nil
	}

	if !a.currentBidIsLessThan(bid.Amount) {
		return ErrBidAmountBelowCurrentBid
	}

	return nil
}

// currentBidIsLessThan returns true if the current bid is less than the bid amount
func (a *Auction) currentBidIsLessThan(bidAmount float64) bool {
	return a.CurrentWinner.Amount < bidAmount
}

// setWinner sets the current winner of the auction
func (a *Auction) setWinner(bid *Bid) {
	a.CurrentWinner = bid
}

// addBidHistory adds a bid to the bid history
func (a *Auction) addBidHistory(bid *Bid) {
	a.Bids = append(a.Bids, *bid)
}

// GetWinner returns the current winner of the auction
func (a *Auction) GetWinner() *Bid {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return a.CurrentWinner
}

// GetBidHistory returns the bid history
func (a *Auction) GetBidHistory() []Bid {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return a.Bids
}
