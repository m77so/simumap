# これはなに
Simutrans用のMapデータを作れます

# 使い方
simumap.exe meishin.json

# jsonファイルの書き方
 - filename
   - 出力する画像ファイルの名前を指定 PNG
 - area
   - 描画する範囲を指定。北端、東端、南端、西端の北緯・東経を度単位で記入
 - drawing
   - 描画方式を指定
   - style
     - Mercator メルカトル図法のみ
   - pixelsize
     - 1ピクセルを何m四方とするか
   - baselat
     - 長さの基準となる緯度を指定
   - margin
     - fill メルカトル図法以外の図法を使用した時に余白をどのように埋めるか
- evelation
  - 標高によりどの明度で着色するかを指定
  - water
    - 海面高度
  - level
    - List
      - min 
      - max
      - bright
      