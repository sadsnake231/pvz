package repoutils

import (
	"fmt"
	"strings"

	"gitlab.ozon.dev/sadsnake2311/homework/internal/domain"
)

func ParsePackaging(input string) (domain.PackagingStrategy, error) {
	types := strings.Split(input, "+")
	var strategies []domain.PackagingStrategy
	var mainPackagingCount int

	for _, t := range types {
		pt := domain.PackagingType(t)

		var strategy domain.PackagingStrategy
		switch pt {
		case domain.PackagingTypePackage:
			strategy = domain.PackagingPackage{}
		case domain.PackagingTypeBox:
			strategy = domain.PackagingBox{}
		case domain.PackagingTypeFilm:
			strategy = domain.PackagingFilm{}
		default:
			return nil, fmt.Errorf("неизвестный тип упаковки")
		}

		if strategy.IsMain() {
			mainPackagingCount++
		}

		if mainPackagingCount > 1 {
			return nil, fmt.Errorf("нельзя комбинировать основные типы упаковки")
		}

		strategies = append(strategies, strategy)
	}

	if len(strategies) == 1 {
		return strategies[0], nil
	}

	return domain.CompositePackaging{Strategies: strategies}, nil
}
