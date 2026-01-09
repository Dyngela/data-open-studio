package pkg

import "api"

func AssertNoError(err error) {
	if err != nil {
		api.Logger.Error().Err(err).Msg("Error occurred that should not have occurred.")
		panic(err)
	}
}
