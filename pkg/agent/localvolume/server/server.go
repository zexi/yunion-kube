package server

import (
	"context"
	"net"
	"os"
	"os/signal"
	"syscall"

	losetup "github.com/zexi/golosetup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
	"k8s.io/utils/exec"
	"k8s.io/utils/mount"

	"yunion.io/x/log"

	pb "yunion.io/x/yunion-kube/pkg/agent/localvolume"
)

type Server struct {
	UnixSocketFile string
}

func NewServer(unixSocketFile string) *Server {
	return &Server{UnixSocketFile: unixSocketFile}
}

func (s *Server) Start() {
	os.Remove(s.UnixSocketFile)
	lis, err := net.Listen("unix", s.UnixSocketFile)
	if err != nil {
		log.Fatalf("listen %s failed: %s", s.UnixSocketFile, err)
	}
	rpcServer := grpc.NewServer()
	pb.RegisterLocalVolumeServer(rpcServer, s)
	// Register reflection service on gRPC server.
	reflection.Register(rpcServer)

	defer func() {
		lis.Close()
		os.Remove(s.UnixSocketFile)
	}()

	go func() {
		if err := rpcServer.Serve(lis); err != nil {
			log.Fatalf("failed to serve rpc: %v", err)
		}
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	// Block until a signal is received
	sig := <-c
	log.Infof("Got signal: %v", sig)
}

func (s *Server) AttachImage(ctx context.Context, in *pb.AttachImageRequest) (*pb.AttachImageResponse, error) {
	diskPath := in.DiskPath
	log.Infof("Attach disk image: %s", diskPath)
	dev, err := losetup.AttachDevice(diskPath)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to attach %s: %v", diskPath, err)
	}
	log.Infof("Disk image %s attached to %s", diskPath, dev.Name)
	return &pb.AttachImageResponse{DevicePath: dev.Name}, nil
}

func (s *Server) DetachDevice(ctx context.Context, in *pb.DetachDeviceRequest) (*pb.DetachDeviceResponse, error) {
	devPath := in.DevicePath
	log.Infof("Detach device: %s", devPath)
	err := losetup.DetachDevice(devPath)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to detach %s: %v", devPath, err)
	}
	log.Infof("Detach device %s success", devPath)
	return &pb.DetachDeviceResponse{}, nil
}

func (s *Server) MountVolume(ctx context.Context, in *pb.MountVolumeRequest) (*pb.MountVolumeResponse, error) {
	targetPath := in.GetTargetPath()
	devPath := in.GetDevicePath()
	fsType := in.GetFsType()
	ro := in.GetReadOnly()
	notMnt, err := mount.New("").IsLikelyNotMountPoint(targetPath)
	if err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(targetPath, 0750); err != nil {
				return nil, status.Errorf(codes.Internal, "Failed to mkdir: %v", err)
			}
			notMnt = true
		} else {
			return nil, status.Errorf(codes.Internal, "Failed to check %s is mountpoint", targetPath, err)
		}
	}

	if !notMnt {
		// already mount to targetPath
		return &pb.MountVolumeResponse{}, nil
	}
	options := []string{}
	if ro {
		options = append(options, "ro")
	}

	diskMounter := &mount.SafeFormatAndMount{Interface: mount.New(""), Exec: exec.New()}
	if err := diskMounter.FormatAndMount(devPath, targetPath, fsType, options); err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to format and mount: %v", err)
	}

	log.Infof("Device %s mounted to volume %s with fs %s", devPath, targetPath, fsType)

	return &pb.MountVolumeResponse{}, nil
}

func (s *Server) UnmountVolume(ctx context.Context, in *pb.UnmountVolumeRequest) (*pb.UnmountVolumeResponse, error) {
	targetPath := in.GetTargetPath()
	mounter := mount.New("")
	notMnt, err := mounter.IsLikelyNotMountPoint(targetPath)

	if err != nil {
		if os.IsNotExist(err) {
			return nil, status.Errorf(codes.NotFound, "Targetpath not found")
		} else {
			return nil, status.Errorf(codes.Internal, "Failed to check %s mountpoint: %v", targetPath, err)
		}
	}
	if notMnt {
		return nil, status.Errorf(codes.NotFound, "Volume %s not mounted", targetPath)
	}

	devicePath, cnt, err := mount.GetDeviceNameFromMount(mounter, targetPath)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to get devicePath from mountpoint %q: %v", targetPath, err)
	}

	// unmounting the image
	err = mounter.Unmount(targetPath)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed unmount %q: %v", targetPath, err)
	}

	cnt--
	log.Infof("Unmounted device %s, ref count: %d", devicePath, cnt)

	return &pb.UnmountVolumeResponse{
		DevicePath: devicePath,
		Count:      int32(cnt),
	}, nil
}

func (s *Server) IsMounted(ctx context.Context, in *pb.IsMountedRequest) (*pb.IsMountedResponse, error) {
	targetPath := in.GetTargetPath()
	mounter := mount.New("")
	notMnt, err := mounter.IsLikelyNotMountPoint(targetPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, status.Errorf(codes.Internal, "Failed to check %s mountpoint: %v", targetPath, err)
		}
	}
	return &pb.IsMountedResponse{
		Mounted: !notMnt,
	}, nil
}
