package pkg

import (
	"encoding/json"
	"fmt"
	"log"
)

func PrettyPrint(T any) {
	message, err := json.MarshalIndent(T, "", "  ")
	if err != nil {
		log.Println("error while printing {}", err)
	}
	fmt.Println(string(message))
}
