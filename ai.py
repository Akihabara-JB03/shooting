import json
import math
import time
import os

# ==============================================================================
# [GitHub Component] Ebiten 2D Bullet Hell Game - Monster Intelligence AI
# Stage Resolution: 320x240 Fixed
# Go Side Sync: Processes orders every 6 frames
# ==============================================================================

# --- 320x240 ステージ専用環境設定 ---
STAGE_WIDTH = 320
STAGE_HEIGHT = 240
CENTER_X = 160.0       # ステージの中心X
CENTER_Y = 120.0       # ステージの中心Y

# Go側の仕様（6フレーム分の移動蓄積）を考慮したセーフティパラメータ
DANGER_WALL_MARGIN = 55       # 壁からこの距離に入ったら強力に中央へ押し戻す
BULLET_DANGER_RADIUS = 35     # 弾の危険感知半径
DAMPING = 0.70                # 慣性の減衰率（壁ハメを防ぐためブレーキを強めに設定）

class EbitenMonsterAI:
    def __init__(self):
        # 過去の移動速度を保持し、滑らかな移動慣性をシミュレート
        self.vx = 0.0
        self.vy = 0.0

    def solve(self):
        # ファイルの存在チェック（I/O競合対策）
        if not os.path.exists("data.json"):
            return

        try:
            with open("data.json", "r") as f:
                content = f.read()
                if not content.strip():
                    return
                data = json.loads(content)
        except Exception:
            return  # Go側の書き込み中などのファイルロック時は安全にスキップ

        # 各種座標・オブジェクトデータの抽出（欠損時はデフォルト値で安全に保護）
        px = data.get("player_x", 0.0)
        py = data.get("player_y", 0.0)
        mx = data.get("mons_x", 0.0)
        my = data.get("mons_y", 0.0)
        bullets = data.get("bullets", [])

        # 🚨 【緊急処理】Go側の制限限界（壁際）に完全に張り付いてしまった場合の救済措置
        # 溜まった慣性を強制リセットし、最優先で中央の安全圏へ引き剥がす出力を生成します
        on_wall = False
        emergency_x, emergency_y = 0.0, 0.0

        if mx <= 40:
            emergency_x = 24.0  # 右方向へ強制脱出
            on_wall = True
        elif mx >= 270:
            emergency_x = -24.0 # 左方向へ強制脱出
            on_wall = True

        if my <= 40:
            emergency_y = 24.0  # 下方向へ強制脱出
            on_wall = True
        elif my >= 190:
            emergency_y = -24.0 # 上方向へ強制脱出
            on_wall = True

        if on_wall:
            self.vx = emergency_x
            self.vy = emergency_y
        else:
            # ――― 通常時の自律思考ルーチン ―――
            avoid_x, avoid_y = 0.0, 0.0

            # 1. 常時働く「ステージ中央への強力な引力」
            # 中央の広いスペースをキープするため、端に寄るほど二乗比例で中心へ戻る力が強くなります
            dx_c = CENTER_X - mx
            dy_c = CENTER_Y - my
            dist_c = math.hypot(dx_c, dy_c)
            if dist_c > 0:
                center_force = (dist_c / 160.0) ** 2
                avoid_x += (dx_c / dist_c) * center_force * 6.5
                avoid_y += (dy_c / dist_c) * center_force * 6.5

            # 2. 角度(Deg)と速度(Speed)に基づく弾道予測回避（未来スキャン）
            if isinstance(bullets, list):
                for b in bullets:
                    if not isinstance(b, dict):
                        continue
                    
                    bx = b.get("X", 0.0)
                    by = b.get("Y", 0.0)
                    deg = b.get("Deg", 0.0)
                    speed = b.get("Speed", 0.0)

                    # 度数法からラジアンに変換して、毎フレームの弾の移動量を割り出す
                    rad = math.radians(deg)
                    b_vx = math.cos(rad) * speed
                    b_vy = math.sin(rad) * speed

                    # 未来の数フレームをシミュレートし、直撃ルートにある弾を検出
                    look_ahead_steps = 4 if speed > 6 else 3
                    for t in range(look_ahead_steps):
                        future_bx = bx + b_vx * t
                        future_by = by + b_vy * t
                        dx = mx - future_bx
                        dy = my - future_by
                        dist = math.hypot(dx, dy)

                        # 自機の弾感知半径に入り、かつ未来で直撃する場合のみ回避行動を取る
                        if 0 < dist < BULLET_DANGER_RADIUS:
                            force = (BULLET_DANGER_RADIUS - dist) / BULLET_DANGER_RADIUS
                            time_weight = 1.0 / (t + 1)
                            # 弾道に対して直角にステップを踏むように力を合成
                            avoid_x += (dx / dist) * force * 15 * time_weight
                            avoid_y += (dy / dist) * force * 15 * time_weight
                            break  # この弾の最危険タイミングを処理したらスキャン終了

            # 3. プレイヤーへの接近と間合い管理（螺旋移動ルーチン）
            dx_p = mx - px
            dy_p = my - py
            dist_p = math.hypot(dx_p, dy_p)
            if dist_p > 0:
                if dist_p < 65:
                    # 近すぎるときは敵っぽく素早くバックステップ
                    avoid_x += (dx_p / dist_p) * 8.0
                    avoid_y += (dy_p / dist_p) * 8.0
                elif dist_p > 130:
                    # 遠いときは執拗にターゲットへ接近
                    avoid_x -= (dx_p / dist_p) * 3.5
                    avoid_y -= (dy_p / dist_p) * 3.5
                else:
                    # 中距離（65〜130）では弾幕の薄い外側へ回り込むように高速旋回
                    orbit_x = -dy_p / dist_p
                    orbit_y = dx_p / dist_p
                    avoid_x += orbit_x * 4.0
                    avoid_y += orbit_y * 4.0

            # 4. 画面の四隅・壁への接近を事前に拒否する斥力
            if mx < DANGER_WALL_MARGIN:
                avoid_x += (DANGER_WALL_MARGIN - mx) * 5.0
            elif mx > STAGE_WIDTH - DANGER_WALL_MARGIN:
                avoid_x -= (mx - (STAGE_WIDTH - DANGER_WALL_MARGIN)) * 5.0

            if my < DANGER_WALL_MARGIN:
                avoid_y += (DANGER_WALL_MARGIN - my) * 5.0
            elif my > STAGE_HEIGHT - DANGER_WALL_MARGIN:
                avoid_y -= (my - (STAGE_HEIGHT - DANGER_WALL_MARGIN)) * 5.0

            # 5. 通常時の移動慣性更新
            self.vx = self.vx * DAMPING + avoid_x
            self.vy = self.vy * DAMPING + avoid_y

        # 計算結果を整数型（int）にして order.json に出力
        order = {"move_x": int(self.vx), "move_y": int(self.vy)}
        try:
            with open("order.json", "w") as f:
                json.dump(order, f)
        except Exception:
            pass

if __name__ == "__main__":
    # PyInstallerでの一本化（EXE化）の際、起動時のCPU負荷スパイクによるフリーズを防ぐウェイト
    time.sleep(0.5)
    
    ai = EbitenMonsterAI()
    while True:
        ai.solve()
        time.sleep(0.015) # 約60fpsの周期でポーリング制御
