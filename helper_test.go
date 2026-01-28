package main

import(
	"testing"
	"fmt"
)

func TestGetVideoAspectRatio(t *testing.T) {
	cases := []struct{
		input string
		expected string
	} {
		{
			input: "./samples/boots-video-horizontal.mp4",
			expected: "16:9",
		},
		{
			input: "./samples/boots-video-vertical.mp4",
			expected: "9:16",
		},
		
	}

	passCount := 0
	failsCount := 0


	for _, c := range cases {
		actual, _ := getVideoAspectRatio(c.input)


		if actual == c.expected {
			passCount++
			fmt.Printf(`--------------------------
Expected:	%s
Actual: 	%s
PASS		
`, c.expected, actual)
		} else {
			failsCount++
			fmt.Printf(`--------------------------
Expected:	%s
Actual: 	%s
Fail		
`, c.expected, actual)
		}
	}
	fmt.Println("--------------------------")
	fmt.Printf("%d passed, %d failed\n", passCount, failsCount)

}


