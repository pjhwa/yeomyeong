package content

import (
	"path/filepath"

	"github.com/pjhwa/yeomyeong/internal/craft"
	"github.com/pjhwa/yeomyeong/internal/economy"
	"github.com/pjhwa/yeomyeong/internal/skill"
	"github.com/pjhwa/yeomyeong/internal/world"
)

// Livelihood is gather nodes, recipes, and the live price book (M3).
type Livelihood struct {
	Craft   *craft.Catalog
	Markets *economy.Book
}

// LoadLivelihood reads content/craft and content/economy. Missing dirs are empty.
func LoadLivelihood(root string, rooms *world.Catalog, items *world.Items, skills *skill.Catalog) (*Livelihood, error) {
	c, err := craft.Load(filepath.Join(root, "craft"), rooms, items, skills)
	if err != nil {
		return nil, err
	}
	m, err := economy.Load(filepath.Join(root, "economy"), items)
	if err != nil {
		return nil, err
	}
	return &Livelihood{Craft: c, Markets: m}, nil
}
