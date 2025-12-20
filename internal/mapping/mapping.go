package mapping

import "sort"

type Mapping struct {
	SourceOffset  int
	ServiceOffset int
	Length        int
}

type MappedLocation struct {
	Offset  int
	Mapping Mapping
}

type MappedRange struct {
	MappedStart  int
	MappedEnd    int
	StartMapping Mapping
	EndMapping   Mapping
}

type Mapper struct {
	Mappings           []Mapping
	sourceOffsetsMemo  *mappingMemo
	serviceOffsetsMemo *mappingMemo
}

func NewMapper(mappings []Mapping) *Mapper {
	return &Mapper{Mappings: mappings}
}

func (m *Mapper) ToSourceRange(serviceStart, serviceEnd int, fallbackToAnyMatch bool) []MappedRange {
	return m.findMatchingStartEnd(serviceStart, serviceEnd, fallbackToAnyMatch, rangeService)
}

func (m *Mapper) ToServiceRange(sourceStart, sourceEnd int, fallbackToAnyMatch bool) []MappedRange {
	return m.findMatchingStartEnd(sourceStart, sourceEnd, fallbackToAnyMatch, rangeSource)
}

func (m *Mapper) ToSourceLocation(serviceOffset int) []MappedLocation {
	return m.findMatchingOffsets(serviceOffset, rangeService)
}

func (m *Mapper) ToServiceLocation(sourceOffset int) []MappedLocation {
	return m.findMatchingOffsets(sourceOffset, rangeSource)
}

type rangeKey int

const (
	rangeSource rangeKey = iota
	rangeService
)

type mappingMemo struct {
	offsets  []int
	mappings [][]int
}

func (m *Mapper) findMatchingOffsets(offset int, fromRange rangeKey) []MappedLocation {
	memo := m.getMemoBasedOnRange(fromRange)
	if len(memo.offsets) == 0 {
		return nil
	}

	start, end, _, _ := binarySearch(memo.offsets, offset)
	toRange := otherRange(fromRange)
	seen := make(map[int]struct{})
	var results []MappedLocation

	for i := start; i <= end; i++ {
		for _, mappingIndex := range memo.mappings[i] {
			if _, ok := seen[mappingIndex]; ok {
				continue
			}
			seen[mappingIndex] = struct{}{}

			mapping := m.Mappings[mappingIndex]
			fromOffset := offsetForRange(mapping, fromRange)
			toOffset := offsetForRange(mapping, toRange)
			mapped, ok := translateOffset(offset, fromOffset, toOffset, mapping.Length, mapping.Length)
			if ok {
				results = append(results, MappedLocation{
					Offset:  mapped,
					Mapping: mapping,
				})
			}
		}
	}

	return results
}

func (m *Mapper) findMatchingStartEnd(
	start int,
	end int,
	fallbackToAnyMatch bool,
	fromRange rangeKey,
) []MappedRange {
	toRange := otherRange(fromRange)
	var mappedStarts []MappedLocation
	var results []MappedRange
	hadMatch := false

	for _, mappedStart := range m.findMatchingOffsets(start, fromRange) {
		mappedStarts = append(mappedStarts, mappedStart)
		mapping := mappedStart.Mapping
		fromOffset := offsetForRange(mapping, fromRange)
		toOffset := offsetForRange(mapping, toRange)
		mappedEnd, ok := translateOffset(end, fromOffset, toOffset, mapping.Length, mapping.Length)
		if ok {
			hadMatch = true
			results = append(results, MappedRange{
				MappedStart:  mappedStart.Offset,
				MappedEnd:    mappedEnd,
				StartMapping: mapping,
				EndMapping:   mapping,
			})
		}
	}

	if !hadMatch && fallbackToAnyMatch {
		if len(mappedStarts) > 0 {
			endMatches := m.findMatchingOffsets(end, fromRange)
			for _, mappedStart := range mappedStarts {
				for _, mappedEnd := range endMatches {
					if mappedEnd.Offset < mappedStart.Offset {
						continue
					}
					results = append(results, MappedRange{
						MappedStart:  mappedStart.Offset,
						MappedEnd:    mappedEnd.Offset,
						StartMapping: mappedStart.Mapping,
						EndMapping:   mappedEnd.Mapping,
					})
					break
				}
			}
		}
	}

	if fallbackToAnyMatch && len(results) == 0 {
		results = append(results, m.findOverlappingRanges(start, end, fromRange)...)
	}

	return results
}

func (m *Mapper) getMemoBasedOnRange(fromRange rangeKey) *mappingMemo {
	if fromRange == rangeSource {
		if m.sourceOffsetsMemo == nil {
			memo := m.createMemo(rangeSource)
			m.sourceOffsetsMemo = &memo
		}
		return m.sourceOffsetsMemo
	}
	if m.serviceOffsetsMemo == nil {
		memo := m.createMemo(rangeService)
		m.serviceOffsetsMemo = &memo
	}
	return m.serviceOffsetsMemo
}

func (m *Mapper) createMemo(key rangeKey) mappingMemo {
	offsetsSet := make(map[int]struct{})
	for _, mapping := range m.Mappings {
		offset := offsetForRange(mapping, key)
		offsetsSet[offset] = struct{}{}
		offsetsSet[offset+mapping.Length] = struct{}{}
	}

	offsets := make([]int, 0, len(offsetsSet))
	for offset := range offsetsSet {
		offsets = append(offsets, offset)
	}
	sort.Ints(offsets)

	mappings := make([][]int, len(offsets))

	for mappingIndex, mapping := range m.Mappings {
		startOffset := offsetForRange(mapping, key)
		endOffset := startOffset + mapping.Length

		startIndex, _, startMatch, startOk := binarySearch(offsets, startOffset)
		endIndex, _, endMatch, endOk := binarySearch(offsets, endOffset)
		if startOk {
			startIndex = startMatch
		}
		if endOk {
			endIndex = endMatch
		}
		if !startOk || !endOk {
			continue
		}

		for i := startIndex; i <= endIndex; i++ {
			mappings[i] = append(mappings[i], mappingIndex)
		}
	}

	return mappingMemo{offsets: offsets, mappings: mappings}
}

func (m *Mapper) findOverlappingRanges(start int, end int, fromRange rangeKey) []MappedRange {
	memo := m.getMemoBasedOnRange(fromRange)
	if len(memo.offsets) == 0 {
		return nil
	}

	startLow, startHigh, _, _ := binarySearch(memo.offsets, start)
	endLow, endHigh, _, _ := binarySearch(memo.offsets, end)
	startIndex := min(startLow, startHigh)
	endIndex := max(endLow, endHigh)
	toRange := otherRange(fromRange)
	seen := make(map[int]struct{})
	var results []MappedRange

	for i := startIndex; i <= endIndex; i++ {
		for _, mappingIndex := range memo.mappings[i] {
			if _, ok := seen[mappingIndex]; ok {
				continue
			}
			seen[mappingIndex] = struct{}{}

			mapping := m.Mappings[mappingIndex]
			fromStart := offsetForRange(mapping, fromRange)
			fromEnd := fromStart + mapping.Length
			if end < fromStart || start > fromEnd {
				continue
			}

			overlapStart := max(start, fromStart)
			overlapEnd := min(end, fromEnd)

			toOffset := offsetForRange(mapping, toRange)
			mappedStart, okStart := translateOffset(overlapStart, fromStart, toOffset, mapping.Length, mapping.Length)
			mappedEnd, okEnd := translateOffset(overlapEnd, fromStart, toOffset, mapping.Length, mapping.Length)
			if !okStart || !okEnd {
				continue
			}

			results = append(results, MappedRange{
				MappedStart:  mappedStart,
				MappedEnd:    mappedEnd,
				StartMapping: mapping,
				EndMapping:   mapping,
			})
		}
	}

	return results
}

func otherRange(key rangeKey) rangeKey {
	if key == rangeSource {
		return rangeService
	}
	return rangeSource
}

func offsetForRange(mapping Mapping, key rangeKey) int {
	if key == rangeSource {
		return mapping.SourceOffset
	}
	return mapping.ServiceOffset
}

func binarySearch(values []int, searchValue int) (low int, high int, match int, hasMatch bool) {
	if len(values) == 0 {
		return 0, -1, 0, false
	}

	low = 0
	high = len(values) - 1

	for low <= high {
		mid := (low + high) / 2
		midValue := values[mid]
		if midValue < searchValue {
			low = mid + 1
		} else if midValue > searchValue {
			high = mid - 1
		} else {
			low = mid
			high = mid
			match = mid
			hasMatch = true
			break
		}
	}

	finalLow := max(min(min(low, high), len(values)-1), 0)
	finalHigh := min(max(max(low, high), 0), len(values)-1)

	return finalLow, finalHigh, match, hasMatch
}

func translateOffset(
	start int,
	fromOffset int,
	toOffset int,
	fromLength int,
	toLengthOptional ...int,
) (int, bool) {
	if start < fromOffset || start > fromOffset+fromLength {
		return 0, false
	}

	toLength := fromLength
	if len(toLengthOptional) > 0 {
		toLength = toLengthOptional[0]
	}

	rangeOffset := min(start-fromOffset, toLength)

	return toOffset + rangeOffset, true
}
