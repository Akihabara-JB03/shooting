package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"math/rand"
	"fmt"
	"os"
	"github.com/gopxl/beep"
	"github.com/gopxl/beep/wav"
	"time"
	"github.com/gopxl/beep/speaker"
	"encoding/json"
	"os/exec"
	"math"
)

type Game struct {
	x int
	y int
	HP int
	kougeki int
	mons Monster
	bullets []bullet

}
type Monster struct {
	monsx int
	monsy int
	monshp int
	wintime int
	kaihuku int
	idouzikan int
	damage int
}
type bullet struct {
	X float64
	Y float64
	Deg int
	Speed int
}
// 構造体を定義（すでにmain内にあるならそれを使います）
type GameData struct {
    PlayerX int      `json:"player_x"`
    PlayerY int      `json:"player_y"`
    MonsX   int      `json:"mons_x"`
    MonsY   int      `json:"mons_y"`
    Bullets []bullet `json:"bullets"` // これを追加！
}

func (g *Game) SaveToJSON() {
    data := GameData{
        PlayerX: g.x,
        PlayerY: g.y,
        MonsX:   g.mons.monsx,
        MonsY:   g.mons.monsy,
        Bullets: g.bullets, // ここにも追加！
    }
    file, _ := json.MarshalIndent(data, "", "  ")
    os.WriteFile("data.json", file, 0644)
}
// 構造体を定義（AIからの指示を受け取る）
type Order struct {
    MoveX int `json:"move_x"`
    MoveY int `json:"move_y"`
}

func (g *Game) ReadOrder() {
    file, err := os.ReadFile("order.json")
    if err != nil {
        return // ファイルがまだない場合は何もしない
    }
    var order Order
    json.Unmarshal(file, &order)

    // AIの指示通りに座標を動かす！
    g.mons.monsx += order.MoveX
    g.mons.monsy += order.MoveY
}


func abs(v int) int {
	if v < 0 { return -v }
	return v
}


func (g *Game)Draw(screen *ebiten.Image) {

	HPmozi := fmt.Sprintf("HP:%d",g.HP)
	MHPmozi := fmt.Sprintf("Monster HP:%d",g.mons.monshp)
	ebitenutil.DebugPrintAt(screen,HPmozi,20,20)
	ebitenutil.DebugPrintAt(screen,MHPmozi,20,50)
	ebitenutil.DebugPrintAt(screen,"P",g.x,g.y)
	ebitenutil.DebugPrintAt(screen,"M",g.mons.monsx,g.mons.monsy)
	// ★ここを追加：弾をすべて "o" で描画する
    for _, b := range g.bullets {
        ebitenutil.DebugPrintAt(screen, "o", int(b.X), int(b.Y))
    }
	if g.mons.monshp == 0 {
		ebitenutil.DebugPrintAt(screen,"You Win!!",130,115)
	}

}

func main() {
	cmd := exec.Command("./ai.exe")
    err := cmd.Start() // AIを裏で走らせる
    if err != nil {
        fmt.Println("AIが起動できなかったよ:", err)
    } else {
        fmt.Println("AIを自動起動しました！")
    }
	// ① WAVファイルを開く
	f, _ := os.Open("bgm.wav")
	
	// ② WAVファイルを解読（デコード）する
	streamer, format, _ := wav.Decode(f)
	
	// ③ 曲を無限ループ（リピート）させる設定にする
	loopStreamer := beep.Loop(-1, streamer)
	_ = speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))
    speaker.Play(loopStreamer)

	game := &Game{
    	x:       100,
    	y:       100,
    	HP:      20,
    	kougeki: 0,
    	mons: Monster{
        	monsx:  80,
        	monsy:  20,
        	monshp: 5000,
    	},
    	bullets: []bullet{}, // 最初は弾がないので空のスライスを指定
	}

	ebiten.RunGame(game)
	cmd.Process.Kill()
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return 320, 240 // ゲーム画面のドット数（横320×縦240）
}
func (g *Game) Update() error {
	if g.mons.monshp <= 0 {
		g.mons.monshp = 0
		g.mons.wintime+=1
		if g.mons.wintime == 300 {
			os.Exit(0)
		}
		return nil
	}
	g.mons.kaihuku += 1
	if g.mons.kaihuku >= 180 {
		g.mons.kaihuku = 0
		g.mons.monshp += rand.Intn(2) + 1
	}
	g.mons.idouzikan += 1
	if g.mons.idouzikan >= 6 {
		g.mons.idouzikan = 0
		g.ReadOrder()
	}
	if g.HP == 0 {
		
		os.Exit(0)
		return nil
	}
	if ebiten.IsKeyPressed(ebiten.KeyA) {
		g.x -= 4
	} else if ebiten.IsKeyPressed(ebiten.KeyS) {
		g.y += 4
	} else if ebiten.IsKeyPressed(ebiten.KeyD) {
		g.x += 4
	} else if ebiten.IsKeyPressed(ebiten.KeyW) {
		g.y -= 4
	} else if ebiten.IsKeyPressed(ebiten.Key1) {
		g.kougeki += 1
		if g.kougeki >= 12 {
			for d,s := 0,2; d < 360 && s<50; d , s= d+15,s+2 {
					newbullets := bullet{X: float64(g.x), Y: float64(g.y), Deg: d, Speed: s}
					g.bullets = append(g.bullets, newbullets)
				
				
			}
			g.kougeki = 0
		}
	} else if ebiten.IsKeyPressed(ebiten.Key2) {
		g.kougeki += 1
		if g.kougeki >= 12 {
			for d,s := 0,5; d < 360 && s<50; d , s= d+25,s+1 {
					newbullets := bullet{X: float64(g.x), Y: float64(g.y), Deg: d, Speed: s}
					g.bullets = append(g.bullets, newbullets)
				
				
			}
			g.kougeki = 0
		}
	}
	for i := range g.bullets {
    	// 角度(deg)からラジアンを計算して位置を更新
    	rad := float64(g.bullets[i].Deg) * math.Pi / 180
    	g.bullets[i].X += math.Cos(rad) * float64(g.bullets[i].Speed)
    	g.bullets[i].Y += math.Sin(rad) * float64(g.bullets[i].Speed)
	}
	var nextBullets []bullet
	for _, b := range g.bullets {
    // 0〜320の範囲内なら生き残り！
		dx := b.X - float64(g.mons.monsx)
        dy := b.Y - float64(g.mons.monsy)
        dist := math.Sqrt(dx*dx + dy*dy)
		if dist <= 20 {
    		g.mons.monshp -= 2
		} else if b.X >= 0 && b.X <= 320 && b.Y >= 0 && b.Y <= 240 {
    		// 画面内 かつ 当たっていない場合だけ残す
    		nextBullets = append(nextBullets, b)
		}
    }
    g.bullets = nextBullets
	// 弾を移動させる処理

	if g.x < 0 { g.x = 0 }
	if g.x > 310 { g.x = 310 } // 320じゃなくて310で止める
	if g.y < 0 { g.y = 0 }
	if g.y > 230 { g.y = 230 } // 240じゃなくて230で止める
	if g.mons.monsx < 0 { g.mons.monsx = 0 }
	if g.mons.monsx > 310 { g.mons.monsx = 310 } // 320じゃなくて310で止める
	if g.mons.monsy < 0 { g.mons.monsy = 0 }
	if g.mons.monsy > 230 { g.mons.monsy = 230 } // 240じゃなくて230で止める
	if abs(g.x-g.mons.monsx) <= 30 && abs(g.y-g.mons.monsy) <= 30 && !(g.x == g.mons.monsx && g.y == g.mons.monsy) {
		g.mons.damage += 1
		if g.mons.damage >= 6 {
			g.mons.damage = 0
			g.HP -= 1
		}
	}
	if g.mons.monshp > 20000 {
		g.mons.monshp = 20000
	}

	g.SaveToJSON()
	return nil
}