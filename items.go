package main

import "github.com/hajimehoshi/ebiten/v2"

type LootItem struct {
	name  string
	value float64
	image *ebiten.Image
}

func GetLootItems() []LootItem {
	var items []LootItem

	wedge_image := LoadImageFromPath("assets/items/wedge.png")
	gear_image := LoadImageFromPath("assets/items/gear.png")
	castle_image := LoadImageFromPath("assets/items/castle.png")
	rock_image := LoadImageFromPath("assets/items/rock.png")
	dagger_image := LoadImageFromPath("assets/items/dagger.png")
	goblet_image := LoadImageFromPath("assets/items/goblet.png")
	bones_image := LoadImageFromPath("assets/items/fish_bones.png")
	crown_image := LoadImageFromPath("assets/items/crown.png")

	wedge := LootItem{"Door Stop", 4, wedge_image}
	gear := LootItem{"Cog", 8, gear_image}
	castle := LootItem{"Sand Castle", 4, castle_image}
	rock := LootItem{"Rock", 5, rock_image}
	dagger := LootItem{"Dagger", 10, dagger_image}
	goblet := LootItem{"Goblet", 15, goblet_image}
	bones := LootItem{"Fish Bones", 2, bones_image}
	crown := LootItem{"Crown", 30, crown_image}

	items = append(items, wedge)
	items = append(items, gear)
	items = append(items, castle)
	items = append(items, rock)
	items = append(items, dagger)
	items = append(items, goblet)
	items = append(items, bones)
	items = append(items, crown)

	return items
}
