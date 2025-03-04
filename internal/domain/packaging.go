package domain

type PackagingType string

const (
	PackagingTypePackage PackagingType = "пакет"
	PackagingTypeBox     PackagingType = "коробка"
	PackagingTypeFilm    PackagingType = "пленка"
)

type PackagingStrategy interface {
	CalculatePrice() float64
	CheckWeight(baseWeight float64) bool
}

// Пакет (5 руб, вес <10 кг)
type PackagingPackage struct{}

func (p PackagingPackage) CalculatePrice() float64         { return 5 }
func (p PackagingPackage) CheckWeight(weight float64) bool { return weight < 10 }

// Коробка (20 руб, вес <30 кг)
type PackagingBox struct{}

func (p PackagingBox) CalculatePrice() float64         { return 20 }
func (p PackagingBox) CheckWeight(weight float64) bool { return weight < 30 }

// Пленка (1 руб, без проверок)
type PackagingFilm struct{}

func (p PackagingFilm) CalculatePrice() float64         { return 1 }
func (p PackagingFilm) CheckWeight(weight float64) bool { return true }

type CompositePackaging struct {
	Strategies []PackagingStrategy
}

func (c CompositePackaging) CalculatePrice() float64 {
	res := 0.0
	for _, s := range c.Strategies {
		res += s.CalculatePrice()
	}
	return res
}
func (c CompositePackaging) CheckWeight(base float64) bool {
	for _, s := range c.Strategies {
		if !s.CheckWeight(base) {
			return false
		}
	}
	return true
}
