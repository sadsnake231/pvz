package domain

type PackagingType string

const(
	PackagingType1 PackagingType = "пакет"
	PackagingType2 PackagingType = "коробка"
	PackagingType3 PackagingType = "пленка"
)


type PackagingStrategy interface {
	CalculatePrice(basePrice float64) float64
	CheckWeight(baseWeight float64) bool
}

// Пакет (+5 руб, вес <10 кг)
type Packaging1 struct{}
func (p Packaging1) CalculatePrice(base float64) float64 { return base + 5 }
func (p Packaging1) CheckWeight(base float64) bool { return base < 10 }

// Коробка (+20 руб, вес <30 кг)
type Packaging2 struct{}
func (p Packaging2) CalculatePrice(base float64) float64 { return base + 20 }
func (p Packaging2) CheckWeight(base float64) bool { return base < 30 }

// Пленка (+1 руб, без проверок)
type Packaging3 struct{}
func (p Packaging3) CalculatePrice(base float64) float64 { return base + 1 }
func (p Packaging3) CheckWeight(base float64) bool { return true }

type CompositePackaging struct {
	Strategies []PackagingStrategy
}
func (c CompositePackaging) CalculatePrice(base float64) float64 {
	for _, s := range c.Strategies {
		base = s.CalculatePrice(base)
	}
	return base
}
func (c CompositePackaging) CheckWeight(base float64) bool {
	for _, s := range c.Strategies {
		if !s.CheckWeight(base) {
			return false
		}
	}
	return true
}
