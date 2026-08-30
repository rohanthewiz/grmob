package main

import (
	. "github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/htmlout"
	"os"
)

func main() {
	ctx := NewContext()
	node := SocialApp().Render(ctx)
	html := htmlout.ExportHTML(node)

	os.WriteFile("social.html", []byte(html), 0644)
}

func SocialApp() View {
	return WithTheme(DefaultTheme,
		SafeArea(
			Column(
				TopBar(),
				Scroll(
					FeedPost("https://picsum.photos/400", "grahms_dev", "Exploring UI in Go!"),
					Spacer(12),
					FeedPost("https://picsum.photos/401", "code_africa", "Sunset in Maputo ❤️"),
					Spacer(12),
					FeedPost("https://picsum.photos/402", "golang.club", "GrMob v0.1 released 🎉"),
				),
				BottomNav(),
			),
		),
	)
}

func TopBar() View {
	return Row(
		BackgroundColor("#FFFFFF"),
		Padding(16),
		Text("GrMobGram", FontSize(20), FontWeight(Bold), TextColor("#000")),
	)
}

func FeedPost(imgURL, username, caption string) View {
	return Card(
		Column(
			Row(
				Image("https://dummyimage.com/40x40/ccc/000&text="+username[:1]),
				Spacer(8),
				Text(username, FontWeight(Bold)),
			),
			Spacer(8),
			Image(imgURL),
			Spacer(8),
			Text(caption, FontSize(14)),
			Spacer(8),
			Row(
				Button("❤️", func() {

				}),
				Spacer(8),
				Button("💬", func() {

				}),
				Spacer(8),
				Button("🔗", func() {

				}),
			),
		),
	)
}

func BottomNav() View {
	return Row(
		BackgroundColor("#FFFFFF"),
		Padding(12),
		Align(AlignJustify),
		Text("🏠"),
		Text("🔍"),
		Text("➕"),
		Text("❤️"),
		Text("👤"),
	)
}
