package main

import (
	"encoding/json"
	"fmt"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/mitchellh/mapstructure"
	"io"
	"log"
	"math"
	"net/http"
)

const (
	singlePortionEggsCount  = 2
	singlePortionFlourGrams = 250
	singlePortionMilkLitres = 0.35
)

func Server() {
	kitchenInput := make(chan interface{}, 10)
	kitchenOutput := make(chan interface{}, 5)
	fryingPan := make(chan struct{}, 3)

	go func() {
		for {
			select {
			case <-fryingPan:
				kitchenOutput <- PancakeReady{}
			}
		}
	}()

	go func() {
		ingredients := Ingredients{}

		for {
			select {
			case inp := <-kitchenInput:
				switch ingredient := inp.(type) {
				case Eggs:
					ingredients.eggs += ingredient.Count
				case Flour:
					ingredients.flour += ingredient.Grams
				case Milk:
					ingredients.milk += ingredient.Litres
				default:
					log.Printf("Unknown ingredient: %v", ingredient)
				}
				fmt.Printf("Ingredients on-hand: %v\n", ingredients)

				kitchenOutput <- IngredientReceived{Ingredient: inp}

				var portionIngredients *PortionIngredients
				if ingredients, portionIngredients = from(ingredients); portionIngredients != nil {
					for i := 0; i < portionIngredients.pancakeCount; i++ {
						fryingPan <- struct{}{}
					}
				}
			}
		}
	}()

	err := http.ListenAndServe(":8080", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, _, _, err := ws.UpgradeHTTP(r, w)
		if err != nil {
			// handle error
		}
		go func() {
			quit := make(chan struct{})

			defer func() {
				_ = conn.Close()
				quit <- struct{}{}
			}()

			var (
				r       = wsutil.NewReader(conn, ws.StateServerSide)
				w       = wsutil.NewWriter(conn, ws.StateServerSide, ws.OpText)
				decoder = json.NewDecoder(r)
				encoder = json.NewEncoder(w)
			)

			go func() {
				for {
					select {
					case result := <-kitchenOutput:
						fmt.Printf("Sending %v\n", result)
						if err := encoder.Encode(result); err != nil {
							log.Print(err)
							return
						}
						_ = w.Flush()

					case <-quit:
						fmt.Println("Quitting...")
						return
					}
				}
			}()

			for {
				hdr, err := r.NextFrame()
				if err != nil {
					if err != io.ErrUnexpectedEOF {
						log.Print(err)
						return
					}
				}
				if hdr.OpCode == ws.OpClose {
					return
				}

				var resp = make(map[string]interface{})
				if err := decoder.Decode(&resp); err != nil {
					log.Print(err)
					return
				}

				switch resp["ingredient"] {
				case "eggs":
					var i Eggs
					if err := mapstructure.Decode(resp, &i); err == nil {
						kitchenInput <- i
					}
				case "flour":
					var i Flour
					if err := mapstructure.Decode(resp, &i); err == nil {
						kitchenInput <- i
					}
				case "milk":
					var i Milk
					if err := mapstructure.Decode(resp, &i); err == nil {
						kitchenInput <- i
					}
				default:
					log.Printf("Unknown ingredient: %v", resp)
				}
			}
		}()
	}))

	log.Fatal(err)
}

type (
	Ingredients struct {
		eggs  int
		flour int
		milk  float64
	}

	PortionIngredients struct {
		ingredients  Ingredients
		pancakeCount int
	}
)

func from(ingredients Ingredients) (Ingredients, *PortionIngredients) {
	portions := int(math.Min(
		float64(ingredients.flour)/singlePortionFlourGrams,
		math.Min(
			ingredients.milk/singlePortionMilkLitres,
			float64(ingredients.eggs)/singlePortionEggsCount,
		),
	))

	var portionIngredients *PortionIngredients

	if portions > 0 {
		pI := Ingredients{
			eggs:  singlePortionEggsCount * portions,
			flour: singlePortionFlourGrams * portions,
			milk:  singlePortionMilkLitres * float64(portions),
		}
		portionIngredients = &PortionIngredients{
			ingredients:  pI,
			pancakeCount: portions,
		}
		ingredients.eggs -= pI.eggs
		ingredients.flour -= pI.flour
		ingredients.milk -= pI.milk
	}

	return ingredients, portionIngredients
}
