# これはなに
Simutrans用のMapデータを作れます

# Install
1. Install Golang
2. Build it. `go build main.go`

# 使い方
## 高度データのダウンロード
1. 下のJSONファイルの書き方を参考に，必要な範囲の設定ファイルを書いてください．とりあえず試したい場合は付属のmeishi.json を使ってみると良いでしょう．
2. NASA のサイトからダウンロードできるように，会員登録を済ませ，適当なデータをダウンロードできることを確かめてください．https://e4ftl01.cr.usgs.gov/MEASURES/SRTMGL3.003/2000.02.11/ 
3. dryrunモードで地形のダウンロードURLの一覧を作成し，適当なテキストに吐きます． ` ./main -d -f meishin.json > urls.txt`
4. ブラウザに一括ダウンロードのアドオンを追加します．Open Multiple Urls はFirefox版もChrome版もあります．
  - FirefoxでSaveボタンを連打しないといけなくて辛いので，Chromeを使うか，ダイアログを止める(https://support.mozilla.org/en-US/questions/1279926)と良いです
5. 一括ダウンロードツールに先程のテキストファイルを入れて，一括ダウンロードします．
6. terrian/ フォルダ以下にコピーします

## マップの作成
1. 高度データをダウンロードします（上述）
2. `./main -f meishin.json`

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
      