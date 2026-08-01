# Execution Lock

## canvas
- viewBox: 0 0 1280 720
- format: PPT 16:9

## colors
- bg: #F7FAFC
- secondary_bg: #EAF2F8
- primary: #0B1F3A
- accent: #1565C0
- secondary_accent: #22C1C3
- warm_accent: #D97706
- text: #1F2937
- text_secondary: #64748B
- text_tertiary: #94A3B8
- border: #CBD5E1
- success: #16A34A
- warning: #D97706
- white: #FFFFFF

## typography
- font_family: &quot;Microsoft YaHei&quot;, &quot;PingFang SC&quot;, Arial, sans-serif
- emphasis_family: Arial, "Microsoft YaHei", sans-serif
- code_family: Consolas, "Courier New", monospace
- body: 20
- title: 34
- subtitle: 25
- annotation: 15
- cover_title: 72
- chapter_title: 44
- hero_number: 42
- footer: 12

## icons
- library: phosphor-duotone
- inventory: robot, users-three, flow-arrow, shield-check, cloud-check, database, chart-line-up, warning-circle, rocket-launch, brain, tree-structure, cpu, gear-six, book-open-text

## page_rhythm
- P01: anchor
- P02: dense
- P03: dense
- P04: breathing
- P05: dense
- P06: dense
- P07: dense
- P08: dense
- P09: breathing
- P10: anchor

## forbidden
- Mixing icon libraries
- rgba()
- `<style>`, `class`, `<foreignObject>`, `textPath`, `@font-face`, `<animate*>`, `<script>`, `<iframe>`, `<symbol>`+`<use>`
- `<g opacity>` (set opacity on each child element individually)
- HTML named entities in text (`&nbsp;`, `&mdash;`, `&copy;`, `&ndash;`, `&reg;`, `&hellip;`, `&bull;`)