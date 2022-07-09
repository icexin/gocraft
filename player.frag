#version 330 core

in vec2 Tex;
uniform sampler2D tex;

out vec4 FragColor;

void main() {
    vec3 color = vec3(texture(tex, vec2(Tex.x, 1-Tex.y)));
    if (color == vec3(1,0,1)) {
        discard;
    }
    FragColor = vec4(color, 1);
}
