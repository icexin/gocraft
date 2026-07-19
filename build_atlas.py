from PIL import Image
import os
import sys

TEXDIR = os.path.join(os.path.dirname(__file__), 'minetest_tex')
PROJDIR = os.path.dirname(__file__)

old_atlas = Image.open(os.path.join(PROJDIR, 'texture.png')).convert('RGBA')
atlas = Image.new('RGBA', (256, 256), (255, 0, 255, 255))

atlas.paste(old_atlas, (0, 0))

def load_tex(name):
    path = os.path.join(TEXDIR, name)
    img = Image.open(path).convert('RGBA')
    if img.size != (16, 16):
        img = img.resize((16, 16), Image.NEAREST)
    return img

def place(tile_idx, img):
    col = tile_idx % 16
    row = tile_idx // 16
    atlas.paste(img, (col * 16, row * 16))

place(0, load_tex('dirt.png'))
place(1, load_tex('sand.png'))
place(2, load_tex('gravel.png'))
place(3, load_tex('stone.png'))
place(4, load_tex('tree_top.png'))
place(5, load_tex('cobblestone.png'))
place(6, load_tex('stone_brick.png'))
place(7, load_tex('brick.png'))
place(8, load_tex('sandstone_bottom.png'))
place(9, load_tex('glass.png'))
place(10, load_tex('diamond.png'))
place(11, load_tex('gold.png'))
place(12, load_tex('iron.png'))
place(13, load_tex('redstone.png'))
place(14, load_tex('leaves.png'))
place(15, load_tex('snow.png'))

place(16, load_tex('grass_side.png'))
place(20, load_tex('tree_side.png'))
place(24, load_tex('sandstone_side.png'))

place(32, load_tex('grass_top.png'))
place(36, load_tex('wood_planks.png'))
place(40, load_tex('sandstone_top.png'))

place(48, load_tex('grass_plant.png'))
place(49, load_tex('flower_dandelion.png'))
place(50, load_tex('flower_rose.png'))
place(51, load_tex('flower_tulip.png'))
place(52, load_tex('flower_viola.png'))
place(53, load_tex('flower_geranium.png'))
place(54, load_tex('flower_mushroom.png'))

old_atlas.save(os.path.join(PROJDIR, 'texture_original.png'))
atlas.save(os.path.join(PROJDIR, 'texture.png'))
print("Atlas saved to texture.png")
print("Original backed up to texture_original.png")
print("Atlas size:", atlas.size)
