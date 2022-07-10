#version 330 core

in vec2 Tex;
in float diff;
in float fog_factor;
uniform sampler2D tex;

out vec4 FragColor;

const vec3 sky_color = vec3(0.57, 0.71, 0.77);

void main() {
    vec3 color = vec3(texture(tex, vec2(Tex.x, 1-Tex.y)));
    if (color == vec3(1,0,1)) {
        discard;
    }
    float df = diff;
    if (color == vec3(1,1,1)) {
        df = 1- diff * 0.2;
    }
    vec3 ambient = 0.5 * vec3(1, 1, 1);
    vec3 diffcolor = df * 0.5 * vec3(1,1,1);
    color = (ambient + diffcolor) * color;
    color = mix(color, sky_color, fog_factor);
    FragColor = vec4(color, 1);
}
