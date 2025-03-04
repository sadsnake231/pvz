package utils_repository

import (
	"fmt"
	"strings"

	"gitlab.ozon.dev/sadsnake2311/homework/internal/domain"
)

func ParsePackaging(input string) (domain.PackagingStrategy, error) {
	types := strings.Split(input, "+")
	strategies := make([]domain.PackagingStrategy, 0, len(types))
	typesConverted := make([]domain.PackagingType, len(types))

	var mainPackagingCount int

	for i, t := range types {
		pt := domain.PackagingType(t)
		typesConverted[i] = pt

		switch pt {
		case domain.PackagingTypePackage, domain.PackagingTypeBox:
			mainPackagingCount++
			if mainPackagingCount > 1 {
				return nil, fmt.Errorf("нельзя комбинировать основные типы упаковки")
			}
		case domain.PackagingTypeFilm:
		default:
			return nil, fmt.Errorf("неизвестный тип упаковки")
		}
	}

	for _, t := range typesConverted {
		switch t {
		case domain.PackagingTypePackage:
			strategies = append(strategies, domain.PackagingPackage{})
		case domain.PackagingTypeBox:
			strategies = append(strategies, domain.PackagingBox{})
		case domain.PackagingTypeFilm:
			strategies = append(strategies, domain.PackagingFilm{})
		}
	}

	if len(strategies) == 1 {
		return strategies[0], nil
	}

	return domain.CompositePackaging{Strategies: strategies}, nil
}
