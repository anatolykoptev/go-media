package dash

import "testing"

// instagramMPD is DERIVED FROM A CAPTURED Instagram DASH response
// (~/tmp/dash-diag/manifest.mpd, captured 2025-07-24). It is byte-faithful to
// the real element/attribute structure that Instagram actually sends:
//
//   - AdaptationSet carries ONLY contentType="video"/"audio" (no slash, no
//     mimeType attribute). The mimeType lives on each Representation.
//   - Each Representation carries FBContentLength (exact byte count).
//   - SegmentBase + Initialization are present; SegmentTemplate is absent.
//   - mediaPresentationDuration="PT47.5S".
//
// A hand-written shape is FORBIDDEN here. The previous synthetic fixture put
// mimeType="video/mp4" on the AdaptationSet — a shape Instagram never sends —
// so the parser's contentType classification bug shipped green: every real
// adaptation set was silently dropped and the extractor degraded to 720p
// video_versions. This fixture exists specifically to prevent that regression.
//
// CDN BaseURL values were sanitized: the real signed host/path/query tokens
// were replaced with identically-shaped fake https://cdn.example.invalid/...
// values. No live signed URLs are committed.
const instagramMPD = `<?xml version="1.0" encoding="UTF-8"?>
<MPD xmlns="urn:mpeg:dash:schema:mpd:2011" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:mpeg:dash:schema:mpd:2011 DASH-MPD.xsd" profiles="urn:mpeg:dash:profile:isoff-on-demand:2011" minBufferTime="PT2S" type="static" mediaPresentationDuration="PT47.5S"><Period id="0" duration="PT47.5S"><AdaptationSet id="0" contentType="video" subsegmentAlignment="true" par="9:16" FBUnifiedUploadResolutionMos="360:81.2"><Representation id="676332058355083v" bandwidth="166515" codecs="vp09.00.31.08.00.01.01.01.00" mimeType="video/mp4" sar="1:1" FBEncodingTag="dash_r2evevp9-r1gen2vp9_q20" FBContentLength="988686" FBPlaybackResolutionMos="0:100,360:70.7,480:70.2,720:70.5,1080:71.9" FBPlaybackResolutionMosConfidenceLevel="high" FBPlaybackResolutionCsvqm="0:100,360:87,480:84.2,720:80,1080:74.3" FBAbrPolicyTags="" width="720" height="1280" frameRate="15360/512" FBQualityClass="hd" FBQualityLabel="240p"><BaseURL>https://cdn.example.invalid/o1/v/t2/f2/m367/FAKE_REP_01.mp4?_nc_cat=100&amp;_nc_sid=FAKE_SID&amp;_nc_ht=cdn.example.invalid&amp;_nc_ohc=FAKE_OHC_01&amp;efg=FAKE_EFG_01&amp;ccb=17-1&amp;_nc_gid=FAKE_GID&amp;_nc_ss=FAKE_SS&amp;_nc_zt=28&amp;oh=FAKE_OH_01&amp;oe=FAKE_OE_01</BaseURL><SegmentBase indexRange="818-969" timescale="15360" FBMinimumPrefetchRange="970-8242" FBPartialPrefetchDuration="2500" FBPartialPrefetchRange="970-34833" FBFirstSegmentRange="970-66241" FBFirstSegmentDuration="5000" FBSecondSegmentRange="66242-143686" FBPrefetchSegmentRange="970-66241" FBPrefetchSegmentDuration="5000"><Initialization range="0-817" /></SegmentBase></Representation><Representation id="2193282264510330v" bandwidth="233752" codecs="vp09.00.31.08.00.01.01.01.00" mimeType="video/mp4" sar="1:1" FBEncodingTag="dash_r2evevp9-r1gen2vp9_q30" FBContentLength="1387908" FBPlaybackResolutionMos="0:100,360:75.3,480:74.5,720:74.5,1080:75.2" FBPlaybackResolutionMosConfidenceLevel="high" FBPlaybackResolutionCsvqm="0:100,360:91.2,480:89.1,720:85.8,1080:80.3" FBAbrPolicyTags="" width="720" height="1280" frameRate="15360/512" FBQualityClass="hd" FBQualityLabel="270p"><BaseURL>https://cdn.example.invalid/o1/v/t2/f2/m367/FAKE_REP_02.mp4?_nc_cat=100&amp;_nc_sid=FAKE_SID&amp;_nc_ht=cdn.example.invalid&amp;_nc_ohc=FAKE_OHC_02&amp;efg=FAKE_EFG_02&amp;ccb=17-1&amp;_nc_gid=FAKE_GID&amp;_nc_ss=FAKE_SS&amp;_nc_zt=28&amp;oh=FAKE_OH_02&amp;oe=FAKE_OE_02</BaseURL><SegmentBase indexRange="818-969" timescale="15360" FBMinimumPrefetchRange="970-9583" FBPartialPrefetchDuration="2500" FBPartialPrefetchRange="970-44493" FBFirstSegmentRange="970-86500" FBFirstSegmentDuration="5000" FBSecondSegmentRange="86501-189793" FBPrefetchSegmentRange="970-86500" FBPrefetchSegmentDuration="5000"><Initialization range="0-817" /></SegmentBase></Representation><Representation id="768619636168185v" bandwidth="344298" codecs="vp09.00.31.08.00.01.01.01.00" mimeType="video/mp4" sar="1:1" FBEncodingTag="dash_r2evevp9-r1gen2vp9_q40" FBContentLength="2044272" FBPlaybackResolutionMos="0:100,360:80.5,480:79.3,720:78.8,1080:78.8" FBPlaybackResolutionMosConfidenceLevel="high" FBPlaybackResolutionCsvqm="0:100,360:94.3,480:92.8,720:90.3,1080:85" FBAbrPolicyTags="" width="720" height="1280" frameRate="15360/512" FBQualityClass="hd" FBQualityLabel="360p"><BaseURL>https://cdn.example.invalid/o1/v/t2/f2/m367/FAKE_REP_03.mp4?_nc_cat=100&amp;_nc_sid=FAKE_SID&amp;_nc_ht=cdn.example.invalid&amp;_nc_ohc=FAKE_OHC_03&amp;efg=FAKE_EFG_03&amp;ccb=17-1&amp;_nc_gid=FAKE_GID&amp;_nc_ss=FAKE_SS&amp;_nc_zt=28&amp;oh=FAKE_OH_03&amp;oe=FAKE_OE_03</BaseURL><SegmentBase indexRange="818-969" timescale="15360" FBMinimumPrefetchRange="970-11663" FBPartialPrefetchDuration="2500" FBPartialPrefetchRange="970-60156" FBFirstSegmentRange="970-122906" FBFirstSegmentDuration="5000" FBSecondSegmentRange="122907-270294" FBPrefetchSegmentRange="970-122906" FBPrefetchSegmentDuration="5000"><Initialization range="0-817" /></SegmentBase></Representation><Representation id="1531980407977808v" bandwidth="468208" codecs="vp09.00.31.08.00.01.01.01.00" mimeType="video/mp4" sar="1:1" FBEncodingTag="dash_r2evevp9-r1gen2vp9_q50" FBContentLength="2779989" FBPlaybackResolutionMos="0:100,360:84.3,480:83.2,720:82.5,1080:82" FBPlaybackResolutionMosConfidenceLevel="high" FBPlaybackResolutionCsvqm="0:100,360:96.1,480:95,720:93,1080:87.8" FBAbrPolicyTags="" width="720" height="1280" frameRate="15360/512" FBQualityClass="hd" FBQualityLabel="480p"><BaseURL>https://cdn.example.invalid/o1/v/t2/f2/m367/FAKE_REP_04.mp4?_nc_cat=100&amp;_nc_sid=FAKE_SID&amp;_nc_ht=cdn.example.invalid&amp;_nc_ohc=FAKE_OHC_04&amp;efg=FAKE_EFG_04&amp;ccb=17-1&amp;_nc_gid=FAKE_GID&amp;_nc_ss=FAKE_SS&amp;_nc_zt=28&amp;oh=FAKE_OH_04&amp;oe=FAKE_OE_04</BaseURL><SegmentBase indexRange="818-969" timescale="15360" FBMinimumPrefetchRange="970-13483" FBPartialPrefetchDuration="2500" FBPartialPrefetchRange="970-83814" FBFirstSegmentRange="970-160935" FBFirstSegmentDuration="5000" FBSecondSegmentRange="160936-358804" FBPrefetchSegmentRange="970-160935" FBPrefetchSegmentDuration="5000"><Initialization range="0-817" /></SegmentBase></Representation><Representation id="1137928938277948v" bandwidth="652277" codecs="vp09.00.31.08.00.01.01.01.00" mimeType="video/mp4" sar="1:1" FBEncodingTag="dash_r2evevp9-r1gen2vp9_q60" FBContentLength="3872895" FBPlaybackResolutionMos="0:100,360:87,480:86,720:85.3,1080:84.4" FBPlaybackResolutionMosConfidenceLevel="high" FBPlaybackResolutionCsvqm="0:100,360:97.2,480:96.5,720:95.1,1080:90.1" FBAbrPolicyTags="" width="720" height="1280" frameRate="15360/512" FBQualityClass="hd" FBQualityLabel="540p"><BaseURL>https://cdn.example.invalid/o1/v/t2/f2/m367/FAKE_REP_05.mp4?_nc_cat=100&amp;_nc_sid=FAKE_SID&amp;_nc_ht=cdn.example.invalid&amp;_nc_ohc=FAKE_OHC_05&amp;efg=FAKE_EFG_05&amp;ccb=17-1&amp;_nc_gid=FAKE_GID&amp;_nc_ss=FAKE_SS&amp;_nc_zt=28&amp;oh=FAKE_OH_05&amp;oe=FAKE_OE_05</BaseURL><SegmentBase indexRange="818-969" timescale="15360" FBMinimumPrefetchRange="970-16010" FBPartialPrefetchDuration="2500" FBPartialPrefetchRange="970-112735" FBFirstSegmentRange="970-218282" FBFirstSegmentDuration="5000" FBSecondSegmentRange="218283-497319" FBPrefetchSegmentRange="970-218282" FBPrefetchSegmentDuration="5000"><Initialization range="0-817" /></SegmentBase></Representation><Representation id="833826883140160v" bandwidth="877797" codecs="vp09.00.31.08.00.01.01.01.00" mimeType="video/mp4" sar="1:1" FBEncodingTag="dash_r2evevp9-r1gen2vp9_q70" FBContentLength="5211922" FBPlaybackResolutionMos="0:100,360:89.1,480:88.1,720:87.2,1080:86" FBPlaybackResolutionMosConfidenceLevel="high" FBPlaybackResolutionCsvqm="0:100,360:98,480:97.5,720:96.3,1080:91.5" FBAbrPolicyTags="" width="720" height="1280" frameRate="15360/512" FBQualityClass="hd" FBQualityLabel="640p"><BaseURL>https://cdn.example.invalid/o1/v/t2/f2/m367/FAKE_REP_06.mp4?_nc_cat=100&amp;_nc_sid=FAKE_SID&amp;_nc_ht=cdn.example.invalid&amp;_nc_ohc=FAKE_OHC_06&amp;efg=FAKE_EFG_06&amp;ccb=17-1&amp;_nc_gid=FAKE_GID&amp;_nc_ss=FAKE_SS&amp;_nc_zt=28&amp;oh=FAKE_OH_06&amp;oe=FAKE_OE_06</BaseURL><SegmentBase indexRange="818-969" timescale="15360" FBMinimumPrefetchRange="970-18957" FBPartialPrefetchDuration="2500" FBPartialPrefetchRange="970-145420" FBFirstSegmentRange="970-300388" FBFirstSegmentDuration="5000" FBSecondSegmentRange="300389-658627" FBPrefetchSegmentRange="970-300388" FBPrefetchSegmentDuration="5000"><Initialization range="0-817" /></SegmentBase></Representation><Representation id="1471991893841319v" bandwidth="1197894" codecs="vp09.00.31.08.00.01.01.01.00" mimeType="video/mp4" sar="1:1" FBEncodingTag="dash_r2evevp9-r1gen2vp9_q80" FBContentLength="7112498" FBPlaybackResolutionMos="0:100,360:90.4,480:89.3,720:88.4,1080:87" FBPlaybackResolutionMosConfidenceLevel="high" FBPlaybackResolutionCsvqm="0:100,360:98.42,480:98.07,720:97.2,1080:92.7" FBAbrPolicyTags="" width="720" height="1280" frameRate="15360/512" FBQualityClass="hd" FBQualityLabel="720p"><BaseURL>https://cdn.example.invalid/o1/v/t2/f2/m367/FAKE_REP_07.mp4?_nc_cat=100&amp;_nc_sid=FAKE_SID&amp;_nc_ht=cdn.example.invalid&amp;_nc_ohc=FAKE_OHC_07&amp;efg=FAKE_EFG_07&amp;ccb=17-1&amp;_nc_gid=FAKE_GID&amp;_nc_ss=FAKE_SS&amp;_nc_zt=28&amp;oh=FAKE_OH_07&amp;oe=FAKE_OE_07</BaseURL><SegmentBase indexRange="818-969" timescale="15360" FBMinimumPrefetchRange="970-23120" FBPartialPrefetchDuration="2500" FBPartialPrefetchRange="970-193142" FBFirstSegmentRange="970-403517" FBFirstSegmentDuration="5000" FBSecondSegmentRange="403518-893223" FBPrefetchSegmentRange="970-403517" FBPrefetchSegmentDuration="5000"><Initialization range="0-817" /></SegmentBase></Representation><Representation id="1700745277266112v" bandwidth="1645195" codecs="vp09.00.40.08.00.01.01.01.00" mimeType="video/mp4" sar="1:1" FBEncodingTag="dash_r2evevp9-r1gen2vp9_q90" FBContentLength="9768348" FBPlaybackResolutionMos="0:100,360:92.2,480:91.1,720:90,1080:89.7" FBPlaybackResolutionMosConfidenceLevel="high" FBPlaybackResolutionCsvqm="0:100,360:98.68,480:98.44,720:97.9,1080:97.2" FBAbrPolicyTags="avoid_on_cellular,avoid_on_cellular_intentional" width="1080" height="1920" frameRate="15360/512" FBQualityClass="hd" FBQualityLabel="960p"><BaseURL>https://cdn.example.invalid/o1/v/t2/f2/m367/FAKE_REP_08.mp4?_nc_cat=100&amp;_nc_sid=FAKE_SID&amp;_nc_ht=cdn.example.invalid&amp;_nc_ohc=FAKE_OHC_08&amp;efg=FAKE_EFG_08&amp;ccb=17-1&amp;_nc_gid=FAKE_GID&amp;_nc_ss=FAKE_SS&amp;_nc_zt=28&amp;oh=FAKE_OH_08&amp;oe=FAKE_OE_08</BaseURL><SegmentBase indexRange="818-969" timescale="15360" FBMinimumPrefetchRange="970-28246" FBPartialPrefetchDuration="2500" FBPartialPrefetchRange="970-269792" FBFirstSegmentRange="970-557065" FBFirstSegmentDuration="5000" FBSecondSegmentRange="557066-1230871" FBPrefetchSegmentRange="970-557065" FBPrefetchSegmentDuration="5000"><Initialization range="0-817" /></SegmentBase></Representation><Representation id="1447419429847706v" bandwidth="2300928" codecs="vp09.00.40.08.00.01.01.01.00" mimeType="video/mp4" sar="1:1" FBEncodingTag="dash_r2evevp9-r1gen2vp9-hfr_q90" FBContentLength="13661765" FBPlaybackResolutionMos="0:100,360:92.6,480:91.7,720:90.8,1080:90.7" FBPlaybackResolutionMosConfidenceLevel="high" FBPlaybackResolutionCsvqm="0:100,360:98.93,480:98.73,720:98.32,1080:97.7" FBAbrPolicyTags="avoid_on_cellular,avoid_on_cellular_intentional" width="1080" height="1920" frameRate="15360/256" FBQualityClass="hd" FBQualityLabel="1080p"><BaseURL>https://cdn.example.invalid/o1/v/t2/f2/m367/FAKE_REP_09.mp4?_nc_cat=100&amp;_nc_sid=FAKE_SID&amp;_nc_ht=cdn.example.invalid&amp;_nc_ohc=FAKE_OHC_09&amp;efg=FAKE_EFG_09&amp;ccb=17-1&amp;_nc_gid=FAKE_GID&amp;_nc_ss=FAKE_SS&amp;_nc_zt=28&amp;oh=FAKE_OH_09&amp;oe=FAKE_OE_09</BaseURL><SegmentBase indexRange="818-969" timescale="15360" FBMinimumPrefetchRange="970-34560" FBPartialPrefetchDuration="2500" FBPartialPrefetchRange="970-391978" FBFirstSegmentRange="970-844568" FBFirstSegmentDuration="5000" FBSecondSegmentRange="844569-1849534" FBPrefetchSegmentRange="970-844568" FBPrefetchSegmentDuration="5000"><Initialization range="0-817" /></SegmentBase></Representation></AdaptationSet><AdaptationSet id="1" contentType="audio" subsegmentStartsWithSAP="1" subsegmentAlignment="true"><Representation id="25268057312789380a" bandwidth="60377" codecs="mp4a.40.5" mimeType="audio/mp4" FBAvgBitrate="60377" audioSamplingRate="44100" FBEncodingTag="dash_ln_heaac_vbr3_audio" FBContentLength="359339" FBPaqMos="88.86" FBAbrPolicyTags=""><AudioChannelConfiguration schemeIdUri="urn:mpeg:dash:23003:3:audio_channel_configuration:2011" value="2" /><BaseURL>https://cdn.example.invalid/o1/v/t2/f2/m86/FAKE_REP_10.mp4?_nc_cat=100&amp;_nc_sid=FAKE_SID&amp;_nc_ht=cdn.example.invalid&amp;_nc_ohc=FAKE_OHC_10&amp;efg=FAKE_EFG_10&amp;ccb=17-1&amp;_nc_gid=FAKE_GID&amp;_nc_ss=FAKE_SS&amp;_nc_zt=28&amp;oh=FAKE_OH_10&amp;oe=FAKE_OE_10</BaseURL><SegmentBase indexRange="824-1143" timescale="44100" FBMinimumPrefetchRange="1144-1487" FBPartialPrefetchDuration="2500" FBPartialPrefetchRange="1144-24725" FBFirstSegmentRange="1144-21071" FBFirstSegmentDuration="2021" FBSecondSegmentRange="21072-38876" FBPrefetchSegmentRange="1144-38876" FBPrefetchSegmentDuration="4017"><Initialization range="0-823" /></SegmentBase></Representation></AdaptationSet></Period></MPD>`

// instagramFixtureDuration matches mediaPresentationDuration in instagramMPD.
const instagramFixtureDuration = 47.5

func TestParseManifestInstagramReal(t *testing.T) {
	man, err := ParseManifest(instagramMPD)
	if err != nil {
		t.Fatalf("ParseManifest: unexpected error: %v", err)
	}

	// The bug: AdaptationSet has only contentType="video"/"audio" (no slash),
	// so the old strings.HasPrefix(mime,"video/") check skipped BOTH sets and
	// ParseManifest returned 0 representations. The fix must classify from
	// contentType OR the Representation-level mimeType.
	if len(man.Videos) != 9 {
		t.Fatalf("Videos: got %d, want 9 (real Instagram manifest has 9 video reps)", len(man.Videos))
	}
	if len(man.Audios) != 1 {
		t.Fatalf("Audios: got %d, want 1", len(man.Audios))
	}
	if man.Duration != instagramFixtureDuration {
		t.Fatalf("Duration = %v, want %v", man.Duration, instagramFixtureDuration)
	}

	// Top video rep must be the 1080x1920 rendition (bandwidth=2300928,
	// FBContentLength=13661765). Find it by dimensions.
	var top *Representation
	for i := range man.Videos {
		if man.Videos[i].Width == 1080 && man.Videos[i].Height == 1920 && man.Videos[i].Bandwidth == 2300928 {
			top = &man.Videos[i]
			break
		}
	}
	if top == nil {
		t.Fatalf("1080x1920 rep not found; videos=%+v", man.Videos)
	}
	if top.ContentLength != 13661765 {
		t.Fatalf("1080p FBContentLength = %d, want 13661765", top.ContentLength)
	}
	if top.URL == "" {
		t.Fatal("1080p URL is empty")
	}
	if top.MimeType != "video/mp4" {
		t.Fatalf("1080p MimeType = %q, want video/mp4 (resolved from Representation-level mimeType)", top.MimeType)
	}

	// Audio rep carries FBContentLength too.
	if man.Audios[0].ContentLength != 359339 {
		t.Fatalf("audio FBContentLength = %d, want 359339", man.Audios[0].ContentLength)
	}
	if man.Audios[0].URL == "" {
		t.Fatal("audio URL is empty")
	}
}

func TestSelectInstagram50MBPicks1080p(t *testing.T) {
	man, err := ParseManifest(instagramMPD)
	if err != nil {
		t.Fatalf("ParseManifest: %v", err)
	}
	// 50 MB budget. The 1080p rep is 13661765 bytes (~13 MB, FBContentLength),
	// well inside the budget. The old code never reached it because parsing
	// returned 0 reps; Select errored and the extractor degraded to 720p.
	video, audio, err := Select(man, 50*1024*1024)
	if err != nil {
		t.Fatalf("Select: unexpected error: %v", err)
	}
	if video.Height != 1920 || video.Width != 1080 {
		t.Fatalf("picked %dx%d, want 1080x1920", video.Width, video.Height)
	}
	if video.URL == "" {
		t.Fatal("chosen video URL is empty")
	}
	if audio.URL == "" {
		t.Fatal("chosen audio URL is empty")
	}
}

func TestSelectInstagramMiddleRungBudget(t *testing.T) {
	// A budget that admits the 720p rung (FBContentLength=7112498) but NOT the
	// 960p rung (FBContentLength=9768348) must pick 720p, never anything larger.
	man, err := ParseManifest(instagramMPD)
	if err != nil {
		t.Fatalf("ParseManifest: %v", err)
	}
	budget := int64(7112498) // exactly fits 720p, excludes 960p (9768348)
	video, _, err := Select(man, budget)
	if err != nil {
		t.Fatalf("Select: %v", err)
	}
	// 720p rep is 720x1280 (height=1280). 960p/1080p are 1080x1920 (height=1920).
	if video.Height != 1280 {
		t.Fatalf("middle-rung budget: picked %dx%d (height %d), want 720x1280 (height 1280)", video.Width, video.Height, video.Height)
	}
}

func TestSelectInstagramFBContentLengthPreferredOverEstimate(t *testing.T) {
	// FBContentLength is the exact byte count and must be preferred over the
	// bandwidth*duration/8 estimate. The 1080p rep: FBContentLength=13661765,
	// estimate = 2300928 * 47.5 / 8 = 13661755 (off by 10 bytes). A budget
	// between the two must still pick 1080p when using the exact size, and
	// must NOT pick 1080p if only the estimate were used.
	man, err := ParseManifest(instagramMPD)
	if err != nil {
		t.Fatalf("ParseManifest: %v", err)
	}
	exact := int64(13661765)
	estimate := int64(float64(2300928) * instagramFixtureDuration / 8) // 13661760
	if exact == estimate {
		t.Fatalf("test setup: exact == estimate (%d), test cannot distinguish", exact)
	}
	// Budget between estimate and exact: estimate < budget < exact.
	// If the selector used the estimate, the 1080p rep (estimate<=budget)
	// would fit and be picked first (highest bandwidth among 1920-height).
	// If it uses the exact FBContentLength, the 1080p rep (exact>budget) does
	// NOT fit; the 960p rep (also 1080x1920, ContentLength 9768348) fits
	// instead and is picked. We assert the EXACT-size behaviour by checking
	// the 1080p rep (unique bandwidth 2300928) is NOT the one picked.
	budget := estimate + 1 // 13661761: > estimate(13661760), < exact(13661765)
	if budget >= exact {
		t.Fatalf("test setup: budget %d >= exact %d", budget, exact)
	}
	video, _, err := Select(man, budget)
	if err != nil {
		t.Fatalf("Select: %v", err)
	}
	if video.Bandwidth == 2300928 {
		t.Fatalf("FBContentLength not preferred: budget=%d (estimate=%d, exact=%d) picked the 1080p rep (bw 2300928), meaning the estimate was used instead of the exact size",
			budget, estimate, exact)
	}
	// Sanity: with the exact size, a budget that DOES fit 1080p picks it.
	video2, _, err := Select(man, exact)
	if err != nil {
		t.Fatalf("Select(exact): %v", err)
	}
	if video2.Bandwidth != 2300928 {
		t.Fatalf("budget==exact(%d) should pick the 1080p rep (bw 2300928), got bw %d", exact, video2.Bandwidth)
	}
}

func TestSelectInstagramFBContentLengthAbsentFallsBackToEstimate(t *testing.T) {
	// When FBContentLength is absent, the bandwidth*duration/8 estimate is
	// still used. Build a manifest with the Instagram shape (contentType on
	// AdaptationSet, mimeType on Representation) but NO FBContentLength, and
	// verify budget selection still works via the estimate.
	const mpd = `<?xml version="1.0" encoding="UTF-8"?>
<MPD xmlns="urn:mpeg:dash:schema:mpd:2011" mediaPresentationDuration="PT0H0M30.000S" type="static">
  <Period>
    <AdaptationSet contentType="video">
      <Representation id="low" width="640" height="360" bandwidth="400000" codecs="avc1.4d401e" mimeType="video/mp4">
        <BaseURL>https://cdn.example.invalid/low.mp4</BaseURL>
      </Representation>
      <Representation id="high" width="1280" height="720" bandwidth="2000000" codecs="avc1.4d401f" mimeType="video/mp4">
        <BaseURL>https://cdn.example.invalid/high.mp4</BaseURL>
      </Representation>
    </AdaptationSet>
    <AdaptationSet contentType="audio">
      <Representation id="aud" bandwidth="64000" codecs="mp4a.40.2" mimeType="audio/mp4">
        <BaseURL>https://cdn.example.invalid/audio.m4a</BaseURL>
      </Representation>
    </AdaptationSet>
  </Period>
</MPD>`
	man, err := ParseManifest(mpd)
	if err != nil {
		t.Fatalf("ParseManifest: %v", err)
	}
	if len(man.Videos) != 2 {
		t.Fatalf("Videos: got %d, want 2", len(man.Videos))
	}
	// estimate(high) = 2000000*30/8 = 7500000. Budget admits low only.
	lowEst := int64(float64(400000) * 30 / 8) // 1500000
	video, _, err := Select(man, lowEst)
	if err != nil {
		t.Fatalf("Select: %v", err)
	}
	if video.Height != 360 {
		t.Fatalf("estimate fallback: picked %dx%d, want 640x360", video.Width, video.Height)
	}
	// And when ContentLength is 0, effectiveSize falls back to estimate.
	for _, v := range man.Videos {
		if v.ContentLength != 0 {
			t.Fatalf("rep %q should have no FBContentLength, got %d", v.ID, v.ContentLength)
		}
	}
}
