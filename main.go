package main

import (
	"fmt"
	"image"
	"image/color"
	"log"
	"math"
	"math/rand"

	"github.com/hajimehoshi/bitmapfont/v3"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

var (
	ground_image            = LoadImageFromPath("assets/ground.png")
	deployed_image          = LoadImageFromPath("assets/deployed.png")
	deployed_gold_image     = LoadImageFromPath("assets/deployed_gold.png")
	not_deployed_image      = LoadImageFromPath("assets/not_deployed.png")
	not_deployed_gold_image = LoadImageFromPath("assets/not_deployed_gold.png")
	jetty_image             = LoadImageFromPath("assets/jetty.png")
	shadow_image            = LoadImageFromPath("assets/shadow.png")
	sell_image              = LoadImageFromPath("assets/sell.png")
	sell_mask_image         = LoadImageFromPath("assets/sell_mask.png")
	bobber_image            = LoadImageFromPath("assets/bobber.png")
	portable_hole_image     = LoadImageFromPath("assets/items/portable_hole.png")
	brush_image             = LoadImageFromPath("assets/items/brush.png")
)

type Game struct {
	width, height  int
	mouseX, mouseY int
	items          []LootItem
	balance        float64
	sand           *ebiten.Image
	sand_copy      *ebiten.Image
	physical_items []*PhysicalLootItem
	in_ui          bool
	shop_items     []ShopItem
}

type ShopItem struct {
	id    string
	label string
	cost  float64
}

type PhysicalLootItem struct {
	position    Vector2
	itemdata    LootItem
	is_dragging bool
	sell_time   int
}

var item_collected LootItem
var (
	rod_deployed         = false
	rod_deploy_vector    = Vector2{0, 0}
	rod_deploy_time      = 0
	rod_deploy_origin    = Vector2{0, 0}
	current_rod_position = Vector2{0, 0}

	is_gold = false

	show_collection_screen = false

	random = rand.New(rand.NewSource(17))

	sand_colour  = color.RGBA{203, 131, 68, 255}
	sand_colours = [4]color.RGBA{
		{231, 167, 72, 255},
		{203, 131, 68, 255},
		{192, 111, 56, 255},
		{170, 90, 60, 255},
	}

	upgrade_value_count = 0
	upgrade_speed_count = 0
	pay_neighbour_count = 0
	upgrade_speed_time  = 3.0

	DEPLOY_TIME = 100

	loot_collected_count  = 0
	letter_sell_tip_shown = false
	letter_shop_tip_shown = false

	show_letter = false
	letter_id   = 0

	end_cutscene  = false
	cutscene_time = 0

	scene = 0
	// 0 -> homescreen
	// 1 -> fishing area
	// 2 -> shop?
)

const (
	SELL_ANIM_TIME = 20
)

func (g *Game) Update() error {
	g.mouseX, g.mouseY = ebiten.CursorPosition()

	if scene == 0 {
		return g.Update0()
	} else if scene == 1 {
		return g.Update1()
	} else if scene == 2 {
		return g.Update2()
	} else {
		panic("Unknown Scene")
	}
}

func (g *Game) Update0() error {
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButton0) {
		scene = 1
		show_letter = true
	}
	return nil
}

func (g *Game) Update1() error {
	in_ui := show_collection_screen || show_letter || end_cutscene
	g.in_ui = in_ui

	for index, item := range g.physical_items {
		if item.sell_time > 1 {
			item.sell_time -= 1
			item.position = Vector2{
				25,
				float64(240-200)*float64(SELL_ANIM_TIME-item.sell_time)/float64(SELL_ANIM_TIME) + 200,
			}
		}
		if item.sell_time == 1 {
			if item.itemdata.name == "portablehole" {
				show_letter = true
				letter_id = 3
				letter_shop_tip_shown = true
			}
			g.balance += item.itemdata.value
			g.physical_items = append(g.physical_items[:index], g.physical_items[index+1:]...) // remove item

			if !letter_shop_tip_shown && g.balance > 50 {
				show_letter = true
				letter_id = 2
				letter_shop_tip_shown = true
			}
			break
		}
	}

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButton0) {
		drag_started := false
		for _, item := range g.physical_items {
			min := item.position
			max := item.position.Add(Vector2{float64(item.itemdata.image.Bounds().Dx()), float64(item.itemdata.image.Bounds().Dy())})
			if item.sell_time == 0 && PointInRect(Vector2{float64(g.mouseX), float64(g.mouseY)}, min, max) {
				item.is_dragging = true
				drag_started = true
				break
			}
		}
		if !rod_deployed && !drag_started && !in_ui {
			rod_deploy_time = 0
			rod_deploy_vector = Vector2{
				float64(g.mouseX) - rod_deploy_origin.x,
				float64(g.mouseY) - rod_deploy_origin.y,
			}
			rod_deployed = true
		}
	} else if ebiten.IsMouseButtonPressed(ebiten.MouseButton0) {
		for _, item := range g.physical_items {
			if item.is_dragging {
				item.position.x = float64(g.mouseX) - float64(item.itemdata.image.Bounds().Max.X)/2
				item.position.y = float64(g.mouseY) - float64(item.itemdata.image.Bounds().Max.Y)/2

				if item.itemdata.name == "portablehole" {
					for _, item := range g.physical_items {
						if item.itemdata.name == "portablehole" || item.sell_time > 0 {
							continue
						}
						min := item.position
						max := item.position.Add(Vector2{float64(item.itemdata.image.Bounds().Dx()), float64(item.itemdata.image.Bounds().Dy())})
						if PointInRect(Vector2{float64(g.mouseX), float64(g.mouseY)}, min, max) {
							item.sell_time = SELL_ANIM_TIME
						}
					}
				} else if item.itemdata.name == "brush" {
					texture := ebiten.NewImage(16, 16)
					texture.Fill(sand_colour)

					if g.mouseX-8 < 0 || g.mouseY-8 < 0 || g.mouseX+8 > g.sand.Bounds().Dx() || g.mouseY+8 > g.sand.Bounds().Dy() {
						continue
					}

					texture.DrawImage(
						ebiten.NewImageFromImage(g.sand_copy.SubImage(image.Rectangle{
							image.Point{int(g.mouseX) - 8, int(g.mouseY) - 8},
							image.Point{int(g.mouseX) + 8, int(g.mouseY) + 8},
						})),
						&ebiten.DrawImageOptions{},
					)

					op := &ebiten.DrawImageOptions{}
					op.GeoM.Translate(float64(g.mouseX)-8, float64(g.mouseY)-8)

					g.sand.DrawImage(texture, op)
				}
			}
		}
	} else if inpututil.IsMouseButtonJustReleased(ebiten.MouseButton0) {
		for _, item := range g.physical_items {
			if item.is_dragging {
				item.is_dragging = false
				if PointInRect(Vector2{float64(g.mouseX), float64(g.mouseY)}, Vector2{20, 193}, Vector2{47, 228}) {
					item.sell_time = SELL_ANIM_TIME
				}
			}
		}
	}

	if rod_deployed {
		rod_deploy_time += 1

		rod_pull_percent := float64(DEPLOY_TIME-rod_deploy_time) / float64(DEPLOY_TIME)

		current_rod_position = rod_deploy_origin.Add(
			rod_deploy_vector.Unit().MultiplyByScalar(rod_deploy_vector.Magnitude() * rod_pull_percent),
		)

		sand_position := Vector2{
			random.Float64()*6 - 3,
			random.Float64()*6 - 3,
		}.Add(current_rod_position)
		col := sand_colours[random.Intn(len(sand_colours))]

		g.sand.Set(int(sand_position.x), int(sand_position.y), col)

	}

	if rod_deployed && rod_deploy_time > DEPLOY_TIME {
		rod_deployed = false
		item_collected = g.items[random.Intn(len(g.items))]
		item_collected.value = item_collected.value + item_collected.value*(float64(upgrade_value_count*10)/100)
		show_collection_screen = true
	}

	if show_letter {
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButton0) {
			show_letter = false
		}
	}

	if show_collection_screen {
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButton0) {
			show_collection_screen = false
			position := Vector2{
				random.Float64()*(float64(g.width)-40) + 20,
				random.Float64()*30 + float64(g.height-30-20),
			}
			g.physical_items = append(g.physical_items, &PhysicalLootItem{position, item_collected, false, 0})

			loot_collected_count += 1

			if !letter_sell_tip_shown && loot_collected_count > 2 {
				show_letter = true
				letter_id = 1
				letter_sell_tip_shown = true
			}
		}
	}

	if end_cutscene {
		cutscene_time += 1
	}

	if ebiten.IsKeyPressed(ebiten.KeyE) {
		scene = 2
	}

	return nil
}

func (g *Game) Update2() error {
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButton0) {
		if g.mouseY > g.height-40 && g.mouseX < 60 {
			scene = 1
		}
	}

	for index, item := range g.shop_items {
		can_buy := g.balance >= item.cost
		if 60+((index-1)*20) < g.mouseY && g.mouseY < 60+((index)*20) {

			if can_buy && inpututil.IsMouseButtonJustPressed(ebiten.MouseButton0) {
				g.balance -= item.cost

				if item.id == "upgrade-value" {
					upgrade_value_count += 1
				} else if item.id == "upgrade-speed" {
					upgrade_speed_count += 1
					upgrade_speed_time -= 0.5
					DEPLOY_TIME = int(upgrade_speed_time * 60)
				} else if item.id == "portable-hole" {
					g.physical_items = append(g.physical_items,
						&PhysicalLootItem{
							Vector2{100, 100},
							LootItem{"portablehole", 0, portable_hole_image},
							false, 0,
						},
					)
				} else if item.id == "item-brush" {
					g.physical_items = append(g.physical_items,
						&PhysicalLootItem{
							Vector2{100, 100},
							LootItem{"brush", 0, brush_image},
							false, 0,
						},
					)
				} else if item.id == "neighbour" {
					pay_neighbour_count += 1

					show_letter = true
					scene = 1
					switch pay_neighbour_count {
					case 1:
						letter_id = 4
					case 2:
						letter_id = 5
					case 3:
						letter_id = 6
					case 4:
						letter_id = 7
					case 5:
						letter_id = 8
					case 6:
						letter_id = 9
					case 7:
						letter_id = 10
					case 8:
						letter_id = 11
					case 9:
						letter_id = 12
					case 10:
						end_cutscene = true
						show_letter = false
					}

				} else if item.id == "gold-rod" {
					if is_gold {
						show_letter = true
						scene = 1
						letter_id = 13
					} else {
						is_gold = true
					}
				}

				updateShop(g)
			}

		}

	}

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	if scene == 0 {
		g.Draw0(screen)
	} else if scene == 1 {
		g.Draw1(screen)
	} else if scene == 2 {
		g.Draw2(screen)
	} else {
		panic("Unknown Scene")
	}
}

func (g *Game) Draw0(screen *ebiten.Image) {
	screen.Fill(color.RGBA{231, 167, 72, 255})

	text.Draw(screen, "Fishing for Gold.", bitmapfont.Face, 5, g.height/2-24, color.White)
	text.Draw(screen, "By Hatchibombotar", bitmapfont.Face, 5, g.height/2-10, color.RGBA{230, 230, 230, 255})
	text.Draw(screen, "click anywhere to start.", bitmapfont.Face, 5, g.height-5, color.White)
}

func (g *Game) Draw1(screen *ebiten.Image) {
	screen.Fill(sand_colour)
	screen.DrawImage(g.sand, &ebiten.DrawImageOptions{})

	if end_cutscene {
		radius := float32(cutscene_time * 2)
		vector.DrawFilledCircle(screen, float32(g.width+5), float32(g.height/2), radius, color.RGBA{44, 104, 195, 255}, false)

		if cutscene_time > 200 {
			text.Draw(screen, "Thank you for playing!\n\nFishing for Gold", bitmapfont.Face, 5, g.height/2-24, color.White)
		}
	}

	screen.DrawImage(ground_image, &ebiten.DrawImageOptions{})
	screen.DrawImage(shadow_image, &ebiten.DrawImageOptions{})
	screen.DrawImage(sell_image, &ebiten.DrawImageOptions{})
	screen.DrawImage(jetty_image, &ebiten.DrawImageOptions{})

	for _, item := range g.physical_items {
		if item.sell_time == 0 {
			continue
		}
		image := item.itemdata.image
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(item.position.Unpack64())

		screen.DrawImage(image, op)
	}

	screen.DrawImage(sell_mask_image, &ebiten.DrawImageOptions{})

	if rod_deployed {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(current_rod_position.Unpack64())
		op.GeoM.Translate(-float64(bobber_image.Bounds().Dx())/2, -float64(bobber_image.Bounds().Dy())/2)
		screen.DrawImage(bobber_image, op)
		var path vector.Path
		path.MoveTo(
			rod_deploy_origin.Unpack32(),
		)
		path.LineTo(
			current_rod_position.Unpack32(),
		)
		path.Close()

		StrokePath(screen, &path, color.RGBA{0, 0, 0, 255}, 3, 0, 0)
		StrokePath(screen, &path, color.RGBA{151, 155, 170, 255}, 1, 0, 0)

		if is_gold {
			screen.DrawImage(deployed_gold_image, &ebiten.DrawImageOptions{})
		} else {
			screen.DrawImage(deployed_image, &ebiten.DrawImageOptions{})
		}
	} else {

		if is_gold {
			screen.DrawImage(not_deployed_gold_image, &ebiten.DrawImageOptions{})
		} else {
			screen.DrawImage(not_deployed_image, &ebiten.DrawImageOptions{})
		}
	}

	for _, item := range g.physical_items {
		if item.sell_time > 0 {
			continue
		}
		image := item.itemdata.image
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(item.position.Unpack64())

		screen.DrawImage(image, op)
	}

	if show_collection_screen {
		black_rect := ebiten.NewImage(120, 55)
		black_rect.Fill(color.RGBA{0, 0, 0, 255})

		text.Draw(black_rect, "Item Collected", bitmapfont.Face, 5, 0, color.White)
		text.Draw(black_rect, item_collected.name, bitmapfont.Face, 5, 20, color.White)
		text.Draw(black_rect, fmt.Sprint("Value: ", item_collected.value), bitmapfont.Face, 5, 40, color.White)

		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(120-16-5, 5)
		black_rect.DrawImage(item_collected.image, op)

		op_rect := &ebiten.DrawImageOptions{}
		op_rect.GeoM.Translate(float64(g.width)/2, float64(g.height)/2)
		op_rect.GeoM.Translate(-float64(black_rect.Bounds().Dx())/2, -float64(black_rect.Bounds().Dy())/2)

		screen.DrawImage(black_rect, op_rect)

		text.Draw(screen, "click anywhere to continue", bitmapfont.Face, 5, g.height-10, color.White)

	}

	if show_letter {
		str := Letters[letter_id]

		letter_container := ebiten.NewImage(screen.Bounds().Dx()-20, screen.Bounds().Dy()-20)
		letter_container.Fill(color.White)

		text.Draw(letter_container, str, bitmapfont.Face, 5, 0, color.Black)

		text.Draw(letter_container, "click anywhere to continue", bitmapfont.Face, 2, letter_container.Bounds().Dy()-5, color.Black)

		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(10, 10)

		screen.DrawImage(letter_container, op)

	}

	if !g.in_ui {
		text.Draw(screen, fmt.Sprint("Balance: ", math.Floor(g.balance*100)/100), bitmapfont.Face, 5, 15, color.White)
	}
}

func (g *Game) Draw2(screen *ebiten.Image) {
	screen.Fill(color.RGBA{231, 167, 72, 255})
	text.Draw(screen, "SHOP", bitmapfont.Face, 5, 20, color.White)

	// text.Draw(screen, "Pay Neighbour - 200", bitmapfont.Face, 5, 60, color.White)

	exit_button := ebiten.NewImage(50, 17)
	exit_button.Fill(color.RGBA{0, 0, 0, 255})
	text.Draw(exit_button, "exit", bitmapfont.Face, 5, exit_button.Bounds().Dy()-5, color.White)

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(5, float64(g.height)-25)
	screen.DrawImage(exit_button, op)

	for index, item := range g.shop_items {
		can_buy := g.balance >= item.cost
		text_colour := color.RGBA{230, 230, 230, 255}
		if 60+((index-1)*20) < g.mouseY && g.mouseY < 60+((index)*20) {

			if can_buy {
				text_colour = color.RGBA{0, 255, 0, 255}
			} else {
				text_colour = color.RGBA{255, 0, 0, 255}
			}

		}
		text.Draw(screen, item.label, bitmapfont.Face, 5, 60+(index*20), text_colour)
	}

	text.Draw(screen, fmt.Sprint("Balance: ", math.Floor(g.balance*100)/100), bitmapfont.Face, 5, 35, color.White)

}

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return 320, 240
}

func main() {
	ebiten.SetWindowSize(640, 480)
	ebiten.SetWindowTitle("Fishing for Gold")
	g := &Game{}
	g.width, g.height = 320, 240
	g.items = GetLootItems()
	g.balance = 0

	sand := ebiten.NewImage(g.width, g.height)
	sand_copy := ebiten.NewImage(g.width, g.height)
	g.sand = sand
	g.sand_copy = sand_copy

	// add pattern to sand
	g.sand.Fill(sand_colour)
	for i := range 50 {
		for x := range g.width {
			y_offset := int(math.Sin(float64(x/4)) * 5)
			g.sand.Set(
				x, (10*i)+y_offset,
				sand_colours[0],
			)
			g.sand.Set(
				x, (10*i)+y_offset+2,
				sand_colours[1],
			)
			g.sand.Set(
				x, (10*i)+y_offset+4,
				sand_colours[2],
			)
		}
	}
	g.sand_copy.DrawImage(sand, &ebiten.DrawImageOptions{})

	updateShop(g)

	rod_deploy_origin = Vector2{160, 149}
	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}

func updateShop(g *Game) {

	neighbour_cost := float64(100 + (pay_neighbour_count * 50))
	g.shop_items = []ShopItem{
		{"neighbour", fmt.Sprint("Pay Neighbour - ", neighbour_cost), neighbour_cost},
	}

	current_upgrade_speed := upgrade_speed_time
	next_upgrade_speed := (upgrade_speed_time - 0.5)
	upgrade_speed_cost := float64(int(50 * (1 + float64(upgrade_speed_count)*0.5)))

	if pay_neighbour_count >= 2 && next_upgrade_speed > 0 {
		g.shop_items = append(g.shop_items,
			ShopItem{"upgrade-speed",
				fmt.Sprint("Decrease Reel Speed (", current_upgrade_speed, " -> ", next_upgrade_speed, ") - ", upgrade_speed_cost),
				upgrade_speed_cost,
			},
		)
	}

	current_upgrade_value := 10 * upgrade_value_count
	next_upgrade_value := 10 * (upgrade_value_count + 1)
	upgrade_value_cost := float64(int(50 * (1 + float64(upgrade_value_count)*0.5)))

	if pay_neighbour_count >= 4 {
		g.shop_items = append(g.shop_items,
			ShopItem{"upgrade-value",
				fmt.Sprint("Increase Loot Value (", current_upgrade_value, "% -> ", next_upgrade_value, "%) - ", upgrade_value_cost),
				upgrade_value_cost,
			},
		)
	}

	if pay_neighbour_count >= 5 {
		g.shop_items = append(g.shop_items, ShopItem{"item-brush", "Brush - 50", 50})
	}

	if pay_neighbour_count >= 6 {
		g.shop_items = append(g.shop_items, ShopItem{"portable-hole", "Portable Hole - 100", 100})
	}

	if pay_neighbour_count >= 8 {
		g.shop_items = append(g.shop_items, ShopItem{"gold-rod", "Golden Rod - 300", 300})
	}
}
