package ui

import "strings"

type digitSegments struct {
	top, upperRight, lowerRight, bottom, lowerLeft, upperLeft, middle bool
}

func segments(char rune) digitSegments {
	const zero, nine, on = '0', '9', '1'
	const digitZero, digitOne, digitTwo, digitThree, digitFour = "1111110", "0110000", "1101101", "1111001", "0110011"
	const digitFive, digitSix, digitSeven, digitEight, digitNine = "1011011", "1011111", "1110000", "1111111", "1111011"
	if char < zero || char > nine {
		return digitSegments{}
	}
	patterns := []string{digitZero, digitOne, digitTwo, digitThree, digitFour, digitFive, digitSix, digitSeven, digitEight, digitNine}
	pattern := patterns[char-zero]
	const top, upperRight, lowerRight, bottom, lowerLeft, upperLeft, middle = 0, 1, 2, 3, 4, 5, 6
	return digitSegments{top: pattern[top] == on, upperRight: pattern[upperRight] == on, lowerRight: pattern[lowerRight] == on, bottom: pattern[bottom] == on, lowerLeft: pattern[lowerLeft] == on, upperLeft: pattern[upperLeft] == on, middle: pattern[middle] == on}
}

func segmentJoin(above, below, right bool) string {
	const blank, topLeft, topRight, bottomLeft, bottomRight, leftTee, rightTee = " ", "┏", "┓", "┗", "┛", "┣", "┫"
	if above && below {
		if right {
			return rightTee
		}
		return leftTee
	}
	if above {
		if right {
			return bottomRight
		}
		return bottomLeft
	}
	if below {
		if right {
			return topRight
		}
		return topLeft
	}
	return blank
}

func (digit digitSegments) row(index int) string {
	const top, middle, bottom, upperEnd = 0, 3, 6, 2
	const horizontal, vertical, endCap, blank, gap = "━━━", "┃", "━", " ", "   "
	if index == top || index == middle || index == bottom {
		lit := digit.top
		leftAbove, leftBelow, rightAbove, rightBelow := false, digit.upperLeft, false, digit.upperRight
		if index == middle {
			lit = digit.middle
			leftAbove, leftBelow, rightAbove, rightBelow = digit.upperLeft, digit.lowerLeft, digit.upperRight, digit.lowerRight
		} else if index == bottom {
			lit = digit.bottom
			leftAbove, leftBelow, rightAbove, rightBelow = digit.lowerLeft, false, digit.lowerRight, false
		}
		if lit {
			return segmentJoin(leftAbove, leftBelow, false) + horizontal + segmentJoin(rightAbove, rightBelow, true)
		}
	}
	left, right := digit.lowerLeft, digit.lowerRight
	if index <= upperEnd {
		left, right = digit.upperLeft, digit.upperRight
	} else if index == middle {
		left, right = digit.upperLeft || digit.lowerLeft, digit.upperRight || digit.lowerRight
	}
	leftText, rightText := blank, blank
	stroke := vertical
	if index == top {
		stroke = endCap
	} else if index == bottom {
		stroke = endCap
	}
	if left {
		leftText = stroke
	}
	if right {
		rightText = stroke
	}
	return leftText + gap + rightText
}

func drawClock(clock string) string {
	const height, zero, upperDot, lowerDot = 7, 0, 2, 4
	const colon, dot, blank, gap, newline = ':', " ● ", "   ", " ", "\n"
	var output strings.Builder
	for row := zero; row < height; row++ {
		for index, char := range clock {
			if index > zero {
				output.WriteString(gap)
			}
			if char == colon {
				text := blank
				if row == upperDot || row == lowerDot {
					text = dot
				}
				output.WriteString(text)
			} else {
				output.WriteString(segments(char).row(row))
			}
		}
		if row < height-1 {
			output.WriteString(newline)
		}
	}
	return output.String()
}
