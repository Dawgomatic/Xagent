# Seedance 2.0 Prompt Recipes

## A) Cinematic Adventure (10s)

```text
Mode: All-Reference
Assets Mapping:
- @image1: first frame and hero appearance
- @video1: camera rhythm
- @audio1: atmosphere pacing

Final Prompt:
9:16 vertical, 10s fantasy adventure cinematic, cel-shading blended with watercolor, cool blue-green palette with warm highlights.
0-3s: hero wakes in a dim ancient chamber; faint glowing runes pulse on wet stone walls; slow dolly out.
3-7s: hero walks to giant rune door and touches circular mechanism; energy ripples activate runes in sequence; heavy door opens into bright light; follow shot.
7-10s: reveal vast world from cliff edge with floating islands and distant glowing ruins; crane up + pullback for scale.
Audio: water-drop echoes and low temple resonance at start; layered activation tones at rune trigger; deep rumble on door opening; orchestral swell on world reveal; wind ambience to end.
Visual control: coherent lighting, physically plausible movement, stable identity.

Negative Constraints:
no watermark, no logo, no subtitles, no on-screen text.
```

## B) Extend Existing Video

```text
Mode: All-Reference
Assets Mapping:
- @video1: source clip to continue
- @image1: style/detail anchor

Final Prompt:
Extend @video1 by 5s. Keep continuity of character identity, outfit, camera direction, and lighting.
0-2s (new segment): continue current motion seamlessly without jump cut.
2-5s (new segment): introduce one escalation action and resolve naturally.
Camera: same lens feel and movement language as @video1.
Audio: continue ambience; add subtle transition swell only at climax.

Negative Constraints:
no watermark, no logo.
```

## C) Replace Character in Existing Video

```text
Mode: All-Reference
Assets Mapping:
- @video1: base choreography and camera
- @image1: replacement character identity

Final Prompt:
Replace the main performer in @video1 with the character from @image1.
Preserve original choreography, timing, transitions, and camera path.
Keep scene composition and lighting style; maintain clean edge blending and stable identity in all frames.

Negative Constraints:
no watermark, no logo, no text.
```

## D) Music Beat-Sync Montage

```text
Mode: All-Reference
Assets Mapping:
- @image1 @image2 @image3 @image4: visual set
- @video1: beat structure and cut rhythm
- @audio1: soundtrack timing reference

Final Prompt:
Create a 12s montage synced to @audio1 beat accents, using @video1 rhythm style.
0-4s: introduce subjects with quick punch-in cuts on beat.
4-8s: alternate medium and close shots with kinetic transitions.
8-12s: crescendo sequence with strongest visual motif, then clean landing frame.
Maintain consistent color grade and dynamic but readable composition.

Negative Constraints:
no watermark, no logo.
```
