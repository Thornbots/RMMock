# -*- coding: utf-8 -*-

import socket
import struct
import os
import time
import sys
import subprocess

# ================= ⚙️ Configuration =================
UDP_IP = "0.0.0.0"
UDP_PORT = 3334
VIDEO_FILENAME = "video_record.hevc"

# Ask for a 20MB buffer
REQUEST_BUF_SIZE = 20 * 1024 * 1024 
# ===============================================

def get_nal_type(payload):
    start_offset = -1
    if payload.startswith(b'\x00\x00\x00\x01'):
        start_offset = 4
    elif payload.startswith(b'\x00\x00\x01'):
        start_offset = 3
    if start_offset == -1 or len(payload) <= start_offset: return -1
    return (payload[start_offset] >> 1) & 0x3F

def main():
    # 1. Set up the network
    sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
    sock.setsockopt(socket.SOL_SOCKET, socket.SO_RCVBUF, REQUEST_BUF_SIZE)
    actual_buf = sock.getsockopt(socket.SOL_SOCKET, socket.SO_RCVBUF)
    
    print(f"\n✅ Receiver started (Live View)")
    print(f"   Buffer: {actual_buf/1024/1024:.2f} MB")
    
    if actual_buf < 5 * 1024 * 1024:
        print("⚠️ Warning: the buffer is too small — run the sudo sysctl command!")

    try:
        sock.bind((UDP_IP, UDP_PORT))
    except OSError:
        print(f"❌ Port {UDP_PORT} is already in use")
        return

    f_video = open(VIDEO_FILENAME, 'wb')

    # 2. Start the ffplay subprocess (pipe mode)
    # We feed whatever Python receives straight into ffplay's mouth
    ffplay_cmd = [
        'ffplay',
        '-window_title', 'Real-time Robot Stream', # window title
        '-f', 'hevc',           # force H.265 format
        '-fflags', 'nobuffer',  # disable input buffering (the key to low latency)
        '-flags', 'low_delay',  # low-latency flag
        '-probesize', '32',     # fastest possible probing
        '-analyzeduration', '0',
        '-sync', 'ext',         # sync to an external clock
        '-i', '-'               # read from stdin
    ]
    
    print("📺 Starting the player window...")
    try:
        # stdin=subprocess.PIPE lets us write data in
        player = subprocess.Popen(ffplay_cmd, stdin=subprocess.PIPE, stderr=subprocess.DEVNULL)
    except FileNotFoundError:
        print("❌ ffplay not found, cannot play. Please install ffmpeg.")
        player = None

    # State variables
    current_frame_id = -1
    current_shards = {}
    
    # Key point: we must wait for an IDR before feeding the pipe, or the picture starts out garbled
    stream_ready = False 
    
    # Cache the parameter sets (VPS/SPS/PPS)
    headers_cache = [] 

    stats_total = 0
    stats_ok = 0
    start_time = time.time()
    last_log = time.time()

    print(f"🚀 Receiving the stream... (press Ctrl+C to stop)")

    try:
        while True:
            data, _ = sock.recvfrom(65535)
            if len(data) <= 8: continue

            header = data[:8]
            # Parse the header
            frame_id, shard_id, _ = struct.unpack('>HHI', header)
            payload = data[8:]

            if frame_id != current_frame_id:
                # Finalize the previous frame
                if current_frame_id != -1 and len(current_shards) > 0:
                    stats_total += 1
                    
                    # Completeness check
                    indices = sorted(current_shards.keys())
                    is_complete = (indices[0] == 0) and ((indices[-1] - indices[0] + 1) == len(indices))

                    if is_complete:
                        full_frame = b''.join([current_shards[k] for k in indices])
                        nal_type = get_nal_type(full_frame)

                        # --- Logic ---
                        is_param = nal_type in [32, 33, 34]
                        is_idr = nal_type in [19, 20, 21]

                        # 1. Always cache the newest parameter packets
                        if is_param:
                            headers_cache.append(full_frame)
                            # Cap the cache size so it cannot grow without bound
                            if len(headers_cache) > 10: headers_cache.pop(0)

                        # 2. Gatekeeper: only open the gate once an IDR arrives
                        # if is_idr:
                        #     stream_ready = True
                        if len(headers_cache) > 0:
                            stream_ready = True
                        
                        # 3. Distribute the data
                        if stream_ready or is_param:
                            # Write to file
                            f_video.write(full_frame)
                            stats_ok += 1
                            
                            # Write to the player (if it is still alive)
                            if player and player.poll() is None:
                                try:
                                    # On an IDR, push the cached SPS/PPS in first so the player does not lose the parameter sets
                                    if is_idr:
                                        for h in headers_cache:
                                            player.stdin.write(h)
                                    
                                    player.stdin.write(full_frame)
                                    player.stdin.flush()
                                except BrokenPipeError:
                                    print("\n⚠️ The player window was closed")
                                    player = None

                # Print the log line (once per second)
                now = time.time()
                if now - last_log >= 1.0:
                    loss = (1 - stats_ok/stats_total)*100 if stats_total > 0 else 0
                    sys.stdout.write(f"\r⏱️ {now-start_time:.0f}s | frames:{stats_total} | complete:{stats_ok} | loss:{loss:.2f}% | state:{'playing' if stream_ready else 'waiting for IDR'} ")
                    sys.stdout.flush()
                    last_log = now

                current_frame_id = frame_id
                current_shards = {}

            current_shards[shard_id] = payload

    except KeyboardInterrupt:
        print("\n🛑 Stopped")
    finally:
        f_video.close()
        sock.close()
        if player:
            player.stdin.close()
            player.terminate()

if __name__ == "__main__":
    main()
