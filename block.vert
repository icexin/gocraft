#version 330 core

in vec3 pos;
in vec2 tex;
in vec3 normal;

uniform mat4 matrix;
uniform vec3 camera;
uniform float fogdis;
uniform vec3 light_dir;

out vec2 Tex;
out float diff;
out float fog_factor;


void main() {
    gl_Position = matrix *  vec4(pos, 1.0);

    float camera_distance = distance(pos, camera);
    fog_factor = pow(clamp(camera_distance/fogdis, 0, 1), 4);
    Tex = tex;
    diff = max(0, dot(normal, normalize(light_dir)));
}
