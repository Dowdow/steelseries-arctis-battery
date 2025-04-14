package main

import "time"

func main() {
	registerGame()
	registerEvent()
	bindEvent()

	batteryPercentages := []int{100, 85, 70, 55, 40, 25, 10}

	for _, percent := range batteryPercentages {
		sendEvent(percent)
		time.Sleep(2 * time.Second)
	}

	// unregisterEvent()
}
