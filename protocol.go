package main

import "encoding/json"

type (
	Eggs struct {
		Count int `json:"count"`
	}
	EggsWrapper Eggs

	Flour struct {
		Grams int `json:"grams"`
	}
	FlourWrapper Flour

	Milk struct {
		Litres float64 `json:"litres"`
	}
	MilkWrapper Milk

	PancakeReady        struct{}
	PancakeReadyWrapper PancakeReady

	IngredientReceived struct {
		Ingredient interface{} `json:"ingredient"`
	}
	IngredientReceivedWrapper IngredientReceived
)

func encode(i interface{}) []byte {
	if res, err := json.Marshal(i); err == nil {
		return res
	}
	return []byte("{}")
}

func (v Eggs) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		EggsWrapper
		Ingredient string `json:"ingredient"`
	}{
		EggsWrapper: EggsWrapper(v),
		Ingredient:  "eggs",
	})
}

func (v Flour) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		FlourWrapper
		Ingredient string `json:"ingredient"`
	}{
		FlourWrapper: FlourWrapper(v),
		Ingredient:   "flour",
	})
}

func (v Milk) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		MilkWrapper
		Ingredient string `json:"ingredient"`
	}{
		MilkWrapper: MilkWrapper(v),
		Ingredient:  "milk",
	})
}

func (v PancakeReady) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		PancakeReadyWrapper
		Status string `json:"status"`
	}{
		PancakeReadyWrapper: PancakeReadyWrapper(v),
		Status:              "pancake-ready",
	})
}

func (v IngredientReceived) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		IngredientReceivedWrapper
		Status string `json:"status"`
	}{
		IngredientReceivedWrapper: IngredientReceivedWrapper(v),
		Status:                    "ingredient-received",
	})
}
