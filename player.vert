#version 330 core

in vec3 pos;
in vec2 tex;
in vec3 normal;

uniform mat4 matrix;

out vec2 Tex;

void main() {
    gl_Position = matrix *  vec4(pos, 1.0);
    Tex = tex;
}
