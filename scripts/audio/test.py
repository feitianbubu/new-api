import base64
with open("alloy.wav", "rb") as f:
    b64 = base64.b64encode(f.read()).decode()
with open('output.txt', 'w') as f:
    f.write(b64)
    