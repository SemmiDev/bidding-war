package main

import (
	"errors"
	"fmt"
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestAuctionTryBid(t *testing.T) {
	// Prepare test data
	startingTime := time.Now().UTC()
	endTime := startingTime.Add(5 * time.Minute)
	item := NewItem("Test Item", 5.0, startingTime, endTime)
	auction := NewAuction(item)

	// Case 1: Valid bid
	bid1 := NewBid("user1@example.com", 10.0)
	err := auction.TryBid(bid1)
	if err != nil {
		t.Errorf("Case 1: Expected no error, got %v", err)
		t.FailNow()
	}

	// Case 2: Bid amount below starting bid
	bid3 := NewBid("user3@example.com", 2.0)
	err = auction.TryBid(bid3)
	if err != nil {
		if !errors.Is(err, ErrBidAmountBelowStartingBid) {
			t.Errorf("Case 2: Expected %v, got %v", ErrBidAmountBelowStartingBid, err)
			t.FailNow()
		}
	}

	// Case 3: Bid amount below current/win bid
	bid2 := NewBid("user4@example.com", 7.0)
	err = auction.TryBid(bid2)
	if err != nil {
		if !errors.Is(err, ErrBidAmountBelowCurrentBid) {
			t.Errorf("Case 3: Expected %v, got %v", ErrBidAmountBelowCurrentBid, err)
			t.FailNow()
		}
	}

	// Case 4: Auction over
	// Simulate an ongoing auction with a short duration for testing
	auction.ChangeTime(time.Now().UTC().Add(1 * time.Millisecond))
	bid4 := NewBid("user4@example.com", 100.0)

	// Simulate a concurrent bid attempt while the auction is ongoing
	var errCh = make(chan error)

	go func() {
		time.Sleep(time.Second * 2) // Simulate a concurrent bid after 0.5 seconds
		err := auction.TryBid(bid4)
		if err != nil && !errors.Is(err, ErrAuctionOver) {
			errCh <- fmt.Errorf("case 4: Expected %v, got %v", ErrAuctionOver, err)
		}

		errCh <- nil
	}()

	// Wait for the goroutine to complete
	err = <-errCh
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	// Verify the winner after the auction ends
	winner := auction.GetWinner()
	if winner == nil {
		t.Errorf("Case 4: Winner verification failed: No winner found")
	}

	// check the winner
	if winner.Email != bid1.Email {
		t.Errorf("Case 4: Winner verification failed: Expected %v, got %v", bid1.Email, winner.Email)
	}
	if winner.Amount != bid1.Amount {
		t.Errorf("Case 4: Winner verification failed: Expected %v, got %v", bid1.Amount, winner.Amount)
	}
	if winner.Timestamp != bid1.Timestamp {
		t.Errorf("Case 4: Winner verification failed: Expected %v, got %v", bid1.Timestamp, winner.Timestamp)
	}
}

// Test case to simulate bidding with many users (1000 users)
func TestSimulateManyUsersBidding(t *testing.T) {
	// Prepare test data
	startingTime := time.Now().UTC()
	endTime := startingTime.Add(10 * time.Second)
	reversePrice := 50.0
	item := NewItem("Many Users Auction", reversePrice, startingTime, endTime)
	auction := NewAuction(item)

	// Simulate 1000 users bidding concurrently
	numUsers := 1000
	var wg sync.WaitGroup
	wg.Add(numUsers)

	errAuctionErrCh := make(chan struct{})

	for i := 0; i < numUsers; i++ {
		go func(userNum int) {
			defer wg.Done()

			if userNum == 20 {
				// the winner
				winnerBidAmount := float64(2000)
				winnerBid := NewBid("sammidev4@gmail.com", winnerBidAmount)
				_ = auction.TryBid(winnerBid)
			}

			// Simulate different bid amounts for each user
			bidAmount := reversePrice + float64(userNum) // from 1 to 1000

			// Create bid for the user
			bid := NewBid("user"+strconv.Itoa(userNum)+"@example.com", bidAmount)

			// Attempt to place a bid
			err := auction.TryBid(bid)
			if err != nil {
				if errors.Is(err, ErrAuctionOver) {
					errAuctionErrCh <- struct{}{}
				}
			}
		}(i)
	}

	// Wait for all bidding goroutines to complete
	wg.Wait()

	// Close the error channel to signal that no more errors will be sent
	close(errAuctionErrCh)

	// Verify the final winner after the auction ends
	winner := auction.GetWinner()
	if winner == nil {
		t.Errorf("Winner verification failed: No winner found")
	}

	if winner.Email != "sammidev4@gmail.com" {
		t.Errorf("Winner verification failed: Expected %v, got %v", "sammidev4@gmail.com", winner.Email)
	}

	// Verify the winner bid amount
	if winner.Amount != 2000 {
		t.Errorf("Winner verification failed: Expected %v, got %v", 2000, winner.Amount)
	}

	fmt.Println("Horray! Auction is over!")
	fmt.Printf("Winner: %v\n", winner.Email)
	fmt.Printf("The history of bids: %v\n", auction.GetBidHistory())
}
