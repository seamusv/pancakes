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
	totalFryingPans         = 3
)

func Server() error {
	err := http.ListenAndServe(":8080", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if conn, _, _, err := ws.UpgradeHTTP(r, w); err == nil {
			quit := make(chan struct{})

			defer func() {
				_ = conn.Close()
				close(quit)
			}()

			kitchenInput := make(chan interface{}, 10)
			kitchenOutput := make(chan interface{}, 5)
			fryingPan := make(chan struct{}, 10)

			for i := 0; i < totalFryingPans; i++ {
				go configureFryingPan(fryingPan, kitchenOutput, quit)
			}

			go configureKitchenPrep(kitchenInput, fryingPan, kitchenOutput, quit)

			go func() {
				for {
					select {
					case result := <-kitchenOutput:
						if err := wsutil.WriteServerText(conn, encode(result)); err != nil {
							log.Print(err)
						}

					case <-quit:
						return
					}
				}
			}()

			for {
				b, err := wsutil.ReadClientText(conn)
				if err != nil {
					if err != io.EOF {
						log.Print(err)
					}
					return
				}

				var resp = make(map[string]interface{})
				if err := json.Unmarshal(b, &resp); err != nil {
					log.Print(err)
					return
				}

				if ingredient, err := convertIngredient(resp); err != nil {
					log.Print(err)
					return
				} else {
					kitchenInput <- ingredient
				}
			}
		}
	}))

	return err
}

func configureFryingPan(fryingPanInput chan struct{}, kitchenOutput chan interface{}, quit chan struct{}) {
	for {
		select {
		case <-fryingPanInput:
			kitchenOutput <- PancakeReady{}

		case <-quit:
			return
		}
	}
}

func configureKitchenPrep(kitchenInput chan interface{}, fryingPanInput chan struct{}, kitchenOutput chan interface{}, quit chan struct{}) {
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

			kitchenOutput <- IngredientReceived{Ingredient: inp}

			var portionIngredients *PortionIngredients
			if ingredients, portionIngredients = processPortions(ingredients); portionIngredients != nil {
				for i := 0; i < portionIngredients.pancakeCount; i++ {
					fryingPanInput <- struct{}{}
				}
			}

		case <-quit:
			return
		}
	}
}

func convertIngredient(resp map[string]interface{}) (interface{}, error) {
	var result interface{}
	var err error
	switch resp["ingredient"] {
	case "eggs":
		var i Eggs
		if err = mapstructure.Decode(resp, &i); err == nil {
			result = i
		}
	case "flour":
		var i Flour
		if err = mapstructure.Decode(resp, &i); err == nil {
			result = i
		}
	case "milk":
		var i Milk
		if err = mapstructure.Decode(resp, &i); err == nil {
			result = i
		}
	default:
		err = fmt.Errorf("unknown ingredient: %v", resp)
	}

	return result, err
}

func processPortions(ingredients Ingredients) (Ingredients, *PortionIngredients) {
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
