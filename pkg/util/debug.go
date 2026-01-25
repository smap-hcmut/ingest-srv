package util

import (
	"encoding/json"
	"fmt"
)

func PrintJson(obj interface{}) {
	prettyJSON, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Println(string(prettyJSON))

}
