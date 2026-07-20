import subprocess, re, sys, os

cwd = os.path.dirname(os.path.abspath(__file__))

def scan_region(region_dir, label):
    result = subprocess.run(
        ['go', 'run', './cmd/mcparse/', region_dir, '--scan'],
        capture_output=True, text=True, cwd=cwd
    )
    blocks = {}
    for line in result.stdout.strip().split('\n'):
        m = re.match(r'\s+(minecraft:\S+):\s+(\d+)', line)
        if m:
            blocks[m.group(1)] = int(m.group(2))
    return blocks

print("Scanning overworld...")
overworld = scan_region('Dungeon-of-Fears/Dungeon of Fears/region', 'overworld')
print("Scanning nether...")
nether = scan_region('Dungeon-of-Fears/Dungeon of Fears/DIM-1/region', 'nether')

all_blocks = {}
for d in [overworld, nether]:
    for k, v in d.items():
        all_blocks[k] = all_blocks.get(k, 0) + v

# Load our mapping from the Go source
result = subprocess.run(
    ['go', 'run', './cmd/mcimport/', '--help'],
    capture_output=True, text=True, cwd=cwd
)

# Read the Go source directly to extract mappings
import_src = open(os.path.join(cwd, 'cmd/mcimport/main.go')).read()

def extract_map(name):
    m = re.search(r'var ' + name + r'\s*=\s*map\[string\](?:int|bool)\{([^}]+)\}', import_src, re.DOTALL)
    if not m:
        return {}
    entries = {}
    for line in m.group(1).split('\n'):
        mm = re.match(r'\s*"([^"]+)":\s*(\S+)', line)
        if mm:
            val = mm.group(2).rstrip(',')
            if val == 'true':
                entries[mm.group(1)] = True
            else:
                try:
                    entries[mm.group(1)] = int(val)
                except:
                    entries[mm.group(1)] = val
    return entries

block_map = extract_map('blockMapping')
plant_map = extract_map('plantMapping')
skip_map = extract_map('skipBlocks')

type_names = {
    0: "air/skip", 1: "grass/dirt", 2: "sand", 3: "gravel",
    4: "stone", 5: "wood/log", 6: "cobblestone", 7: "stone_brick",
    8: "brick", 9: "sandstone", 10: "glass", 11: "diamond",
    12: "gold", 13: "iron", 14: "redstone/lava", 15: "leaves",
    16: "cloud", 17: "grass_plant", 18: "dandelion", 19: "poppy",
    20: "cornflower", 21: "azure_bluet", 22: "oxeye_daisy",
    23: "mushroom/wither_rose",
}

properly_mapped = {}
fallback_mapped = {}
skipped = {}
unmapped_default = {}

for name, count in sorted(all_blocks.items(), key=lambda x: -x[1]):
    if name in skip_map:
        skipped[name] = count
        continue
    if name.endswith('_air') or name == 'minecraft:air':
        skipped[name] = count
        continue
    if name in block_map:
        tid = block_map[name]
        properly_mapped[name] = (count, type_names.get(tid, f"type_{tid}"), tid)
        continue
    if name in plant_map:
        tid = plant_map[name]
        properly_mapped[name] = (count, type_names.get(tid, f"type_{tid}"), tid)
        continue
    # Fallback heuristic
    if 'stone' in name or 'brick' in name:
        fallback_mapped[name] = (count, "stone_brick(fallback)", 7)
    elif 'wood' in name or 'log' in name or 'plank' in name:
        fallback_mapped[name] = (count, "wood(fallback)", 5)
    elif 'sand' in name:
        fallback_mapped[name] = (count, "sand(fallback)", 2)
    elif 'dirt' in name or 'grass' in name:
        fallback_mapped[name] = (count, "grass(fallback)", 1)
    elif 'leaves' in name:
        fallback_mapped[name] = (count, "leaves(fallback)", 15)
    elif 'iron' in name or 'metal' in name:
        fallback_mapped[name] = (count, "iron(fallback)", 13)
    elif 'glass' in name:
        fallback_mapped[name] = (count, "glass(fallback)", 10)
    else:
        unmapped_default[name] = (count, "stone(default)", 4)

total = sum(all_blocks.values())
total_air = sum(v for k, v in all_blocks.items() if 'air' in k)
total_solid = total - total_air
mapped_n = sum(v[0] for v in properly_mapped.values())
fallback_n = sum(v[0] for v in fallback_mapped.values())
skipped_n = sum(v for v in skipped.values())
unmapped_n = sum(v[0] for v in unmapped_default.values())

print()
print("=" * 70)
print("DUNGEON OF FEARS - FULL BLOCK MAPPING ANALYSIS")
print("(Overworld + Nether combined)")
print("=" * 70)
print(f"Total block entries:  {total:>10,}")
print(f"Air/cave_air:         {total_air:>10,}")
print(f"Non-air blocks:       {total_solid:>10,}")
print()
pct = lambda n: f"{n*100/total_solid:.1f}%" if total_solid else "0%"
print(f"Properly mapped:      {mapped_n:>10,}  ({pct(mapped_n)})")
print(f"Fallback heuristic:   {fallback_n:>10,}  ({pct(fallback_n)})")
print(f"Skipped (invisible):  {skipped_n:>10,}")
print(f"Unmapped (->stone):   {unmapped_n:>10,}  ({pct(unmapped_n)})")
print()

print("=" * 70)
print("FALLBACK-MAPPED BLOCKS (reasonable guess, not exact texture)")
print("=" * 70)
for name, (count, cat, tid) in sorted(fallback_mapped.items(), key=lambda x: -x[1][0]):
    print(f"  {count:>6}x  {name} -> {cat}")

print()
print("=" * 70)
print("UNMAPPED BLOCKS (all become generic stone)")
print("=" * 70)
for name, (count, cat, tid) in sorted(unmapped_default.items(), key=lambda x: -x[1][0]):
    print(f"  {count:>6}x  {name} -> {cat}")

print()
print("=" * 70)
print("BIGGEST TEXTURE ACCURACY ISSUES")
print("(blocks that map to WRONG-LOOKING textures)")
print("=" * 70)
issues = [
    ("minecraft:netherrack", "stone(4)", "needs red/brown nether texture"),
    ("minecraft:soul_sand", "sand(2)", "needs dark brown, not yellow"),
    ("minecraft:soul_soil", "sand(2)", "needs dark brown"),
    ("minecraft:basalt", "stone(4)", "needs dark gray columnar texture"),
    ("minecraft:blackstone", "stone(4)", "needs very dark stone"),
    ("minecraft:polished_blackstone", "stone(4)", "needs dark polished stone"),
    ("minecraft:deepslate", "stone(4)", "needs dark layered stone"),
    ("minecraft:cobbled_deepslate", "stone(4)", "needs dark cobbled stone"),
    ("minecraft:deepslate_bricks", "stone(4)", "needs dark brick pattern"),
    ("minecraft:obsidian", "stone(4)", "needs dark purple/black"),
    ("minecraft:crimson_stem", "wood(5)", "needs red/crimson wood"),
    ("minecraft:crimson_planks", "wood(5)", "needs red planks"),
    ("minecraft:warped_stem", "wood(5)", "needs teal/warped wood"),
    ("minecraft:nether_wart_block", "leaves(15)", "needs dark red texture"),
    ("minecraft:red_nether_bricks", "brick(8)", "needs dark red brick"),
    ("minecraft:lava", "redstone(14)", "needs orange/flowing lava"),
    ("minecraft:water", "glass(10)", "needs blue transparent"),
    ("minecraft:iron_bars", "iron(13)", "needs thin bar texture"),
    ("minecraft:chain", "iron(13)", "needs chain link texture"),
    ("minecraft:glowstone", "???", "needs yellow/glowing block"),
    ("minecraft:magma_block", "???", "needs orange/cracked lava"),
    ("minecraft:nether_quartz_ore", "???", "needs quartz speckled netherrack"),
    ("minecraft:nether_gold_ore", "???", "needs gold speckled netherrack"),
    ("minecraft:ancient_debris", "???", "needs dark brown/spiral texture"),
    ("minecraft:shroomlight", "???", "needs orange/yellow organic glow"),
    ("minecraft:crimson_nylium", "grass(1)", "needs red nylium top"),
    ("minecraft:warped_nylium", "grass(1)", "needs teal nylium top"),
    ("minecraft:polished_blackstone_bricks", "brick(8)", "needs very dark brick"),
    ("minecraft:purpur_pillar", "brick(8)", "needs purple/pink pillar"),
    ("minecraft:red_sandstone", "sandstone(9)", "needs red sandstone"),
    ("minecraft:hay_block", "sandstone(9)", "needs yellow hay bale"),
    ("minecraft:moss_block", "???", "needs green mossy texture"),
]
for name, maps_to, issue in issues:
    if name in all_blocks:
        print(f"  {all_blocks[name]:>6}x  {name} -> {maps_to}: {issue}")

print()
print("=" * 70)
print("SUGGESTED NEW TEXTURES TO ADD (sorted by impact)")
print("=" * 70)
suggestions = {}
for name, maps_to, issue in issues:
    if name in all_blocks:
        base = name.replace("minecraft:", "").split("_")[0]
        key = issue.split(" ")[1] if "needs" in issue else name
        if key not in suggestions or all_blocks[name] > suggestions[key][1]:
            suggestions[key] = (name, all_blocks[name], issue)

for key, (name, count, issue) in sorted(suggestions.items(), key=lambda x: -x[1][1]):
    print(f"  {count:>6}x  {issue}")
