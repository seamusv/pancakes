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
	"time"
)

const (
	singlePortionEggsCount  = 2
	singlePortionFlourGrams = 250
	singlePortionMilkLitres = 0.35
	totalFryingPans         = 3
)

func Server() error {
	kitchenInput := make(chan interface{}, 10)
	kitchenOutput := make(chan interface{}, 5)
	fryingPan := make(chan struct{}, 3)

	for i := 0; i < totalFryingPans; i++ {
		go configureFryingPan(fryingPan, kitchenOutput)
	}

	go configureKitchenPrep(kitchenInput, fryingPan, kitchenOutput)

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
						if err := encoder.Encode(result); err != nil {
							log.Print(err)
							return
						}
						_ = w.Flush()

					case <-quit:
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

				if ingredient, err := convertIngredient(resp); err != nil {
					log.Print(err)
					return
				} else {
					kitchenInput <- ingredient
				}
			}
		}()
	}))

	return err
}

func configureFryingPan(fryingPanInput chan struct{}, kitchenOutput chan interface{}) {
	for {
		select {
		case <-fryingPanInput:
			time.Sleep(time.Second * 2)
			kitchenOutput <- PancakeReady{}
		}
	}
}

func configureKitchenPrep(kitchenInput chan interface{}, fryingPanInput chan struct{}, kitchenOutput chan interface{}) {
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
