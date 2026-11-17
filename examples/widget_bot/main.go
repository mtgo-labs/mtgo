// Command widget_bot demonstrates the Bot API 10.4 HTML-widget surface:
// sending a widget, editing its state in place, and answering the widget's
// requests live.
//
// Widgets are served by the Telegram TEST server today (Config.TestMode),
// and are rolling out to production for private chats in steps.
//
// Setup (test server):
//
//  1. Log into the test environment (Telegram Desktop: Settings →
//     Shift+Alt+Right-click "Add Account" → Test Server) and register a bot
//     with the test @BotFather.
//
//  2. /start the bot from your test account and note your test user id.
//
//  3. Run:
//
//     BOT_TOKEN=123:abc TEST=1 CHAT_ID=<your test user id> go run ./examples/widget_bot
//
// Then send /widget to the bot and tap the counter inside the bubble.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/mtgo-labs/mtgo/telegram"
	"github.com/mtgo-labs/mtgo/telegram/params"
)

// counterHTML is a minimal interactive widget: a tap sends the "inc"
// request to the bot, which answers with the new value and also edits the
// message's public data so every copy of the widget follows along.
// The bridge (window.TelegramWidget) is provided by the host app.
const counterHTML = `<!doctype html><meta charset="utf-8">
<style>
body{margin:0;padding:14px;font:15px system-ui,sans-serif;color:var(--tg-theme-text-color);
display:grid;place-items:center;gap:10px}
#n{font-size:44px;font-weight:700;font-variant-numeric:tabular-nums}
button{border:0;border-radius:10px;padding:10px 26px;font:600 15px system-ui;
background:var(--tg-theme-button-color);color:var(--tg-theme-button-text-color);cursor:pointer}
button:disabled{opacity:.6}
#s{font-size:12px;color:var(--tg-theme-hint-color)}
</style>
<div id="n">0</div>
<button id="b">tap me</button>
<div id="s"></div>
<script>
const T=window.TelegramWidget,$=i=>document.getElementById(i);
const P={n:0};
const load=()=>{try{P.n=JSON.parse(T&&T.publicData).n||0}catch(e){}$('n').textContent=P.n};
const say=t=>{$('s').textContent=t;setTimeout(()=>$('s').textContent='',2500)};
$('b').onclick=()=>{
  if(!T){say('no bridge');return}
  $('b').disabled=true;
  T.request('inc').then(a=>{
    if(a&&typeof a.n==='number'){P.n=a.n;$('n').textContent=P.n;say('from the bot')}
  },()=>say('request failed')).finally(()=>$('b').disabled=false);
};
if(T){
  T.onEvent('publicDataChanged',load);
  T.onEvent('activated',load);
}
load();
</script>`

type counterState struct {
	N int `json:"n"`
}

func main() {
	token := os.Getenv("BOT_TOKEN")
	if token == "" {
		log.Fatal("BOT_TOKEN is required")
	}
	chatID, err := strconv.ParseInt(os.Getenv("CHAT_ID"), 10, 64)
	if err != nil {
		log.Fatal("CHAT_ID (the peer to send the widget to) is required")
	}
	testMode := os.Getenv("TEST") == "1"

	client, err := telegram.NewClient(2040, "b18441a1ff607e10a989891a5462e627", &telegram.Config{
		BotToken: token,
		TestMode: testMode,
		InMemory: true,
	})
	if err != nil {
		log.Fatal(err)
	}
	if err := client.Connect(30 * time.Second); err != nil {
		log.Fatal("connect:", err)
	}
	defer client.Disconnect()

	me, err := client.GetMe(context.Background())
	if err != nil {
		log.Fatal("getMe:", err)
	}
	log.Printf("authorized as @%s (test=%v)", me.Username, testMode)

	state := counterState{N: 0}
	pub := func() string { b, _ := json.Marshal(state); return string(b) }

	// The widget's requests arrive as regular updates.
	client.AddHandler(&widgetHandler{client: client, state: &state, pub: pub, chatID: chatID})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	msg, err := client.SendWidget(ctx, chatID, 0, 0, counterHTML, &params.SendWidget{
		AllowQueries: true,
		PublicData:   pub(),
		ChatData:     `{"hello":"world"}`,
		Caption:      "mtgo counter widget",
	})
	if err != nil {
		log.Fatal("sendWidget:", err)
	}
	log.Printf("widget sent (message %d) — tap the button in the app", msg.ID)

	// Edit the state once to demonstrate the in-place update.
	time.Sleep(3 * time.Second)
	if _, err := client.EditMessageWidgetData(ctx, chatID, msg.ID, pub(), ""); err != nil {
		log.Printf("editMessageWidgetData: %v", err)
	} else {
		log.Print("widget state edited in place")
	}

	log.Print("listening for widget queries for 10 minutes…")
	<-ctx.Done()
}

type widgetHandler struct {
	client *telegram.Client
	state  *counterState
	pub    func() string
	chatID int64
}

func (h *widgetHandler) Check(u *telegram.Update) bool {
	return u.WidgetQuery != nil
}

func (h *widgetHandler) Handle(ctx *telegram.Context) {
	q := ctx.WidgetQuery
	if string(q.Data) != "inc" {
		_ = h.client.AnswerWidgetQuery(ctx.Ctx, q.ID, `{"err":"unknown request"}`)
		return
	}
	h.state.N++
	// The edit makes every copy of the widget follow the new value;
	// the answer satisfies the tap that asked.
	if _, err := h.client.EditMessageWidgetData(ctx.Ctx, q.ChatID, q.MessageID, h.pub(), ""); err != nil {
		log.Printf("edit: %v", err)
	}
	answer, _ := json.Marshal(map[string]int{"n": h.state.N})
	if err := h.client.AnswerWidgetQuery(ctx.Ctx, q.ID, string(answer)); err != nil {
		log.Printf("answer: %v", err)
		return
	}
	fmt.Printf("widget query from user %d: counter → %d\n", q.UserID, h.state.N)
}
