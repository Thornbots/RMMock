import socket

def udp_listener(port, buffer_size=1024):
    """
    Listens on the given UDP port, receives packets, and prints information about them.

    Args:
        port (int): the port number to listen on.
        buffer_size (int): the size of the receive buffer, in bytes.
    """
    # Create a UDP socket
    try:
        # socket.AF_INET means use IPv4
        # socket.SOCK_DGRAM means use UDP
        sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
    except Exception as e:
        print(f"❌ Error: failed to create the socket: {e}")
        return

    # Bind to the given port on every available interface
    server_address = ('0.0.0.0', port)
    try:
        sock.bind(server_address)
    except Exception as e:
        print(f"❌ Error: failed to bind to port {port}: {e}")
        # Try to close the socket to release the resource
        sock.close()
        return

    print(f"👂 Listening on UDP port {port}...")
    print("Waiting for packets... (press Ctrl+C to stop)")

    try:
        while True:
            # Receive a packet. data is the received bytes, address is the sender's address.
            data, address = sock.recvfrom(buffer_size)

            packet_length = len(data)

            # Print the packet length
            print("-" * 30)
            print(f"📦 Received a packet from {address}.")
            print(f"📏 Total packet length: {packet_length} bytes")

            # Extract and print the header
            print(f"Header: {int.from_bytes(data[0:2],'big',signed=False)} {int.from_bytes(data[2:4],'big',signed=False)} {int.from_bytes(data[4:8],'big',signed=False)} ")

    except KeyboardInterrupt:
        # The user pressed Ctrl+C to stop the program
        print("\n✋ Program stopped.")
    except Exception as e:
        print(f"\n❌ Exception: {e}")
    finally:
        # Close the socket
        sock.close()
        print(f"Stopped listening on port {port}.")

# --- Entry point ---
if __name__ == "__main__":
    # Change this to whichever port you want to listen on
    LISTEN_PORT = 3334
    udp_listener(LISTEN_PORT)