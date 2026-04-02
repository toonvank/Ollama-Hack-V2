import React from "react";
import {
  Modal,
  ModalContent,
  ModalHeader,
  ModalBody,
  ModalFooter,
} from "@heroui/modal";
import { Button } from "@heroui/button";

export type DialogType = "error" | "confirm";

interface DialogModalProps {
  isOpen: boolean;
  onClose: () => void;
  title?: string;
  message: string;
  type: DialogType;
  onConfirm?: () => void;
}

/**
 * Dialog component supporting error alerts and confirmation dialogs
 */
const DialogModal: React.FC<DialogModalProps> = ({
  isOpen,
  onClose,
  title,
  message,
  type,
  onConfirm,
}) => {
  const defaultTitle = type === "error" ? "Operation Failed" : "Confirm Action";

  return (
    <Modal
      hideCloseButton={false}
      isDismissable={type === "error"}
      isOpen={isOpen}
      placement="center"
      size="sm"
      onClose={onClose}
    >
      <ModalContent>
        <ModalHeader
          className={`flex flex-col gap-1 ${type === "error" ? "text-red-600" : type === "confirm" ? "text-blue-600" : ""}`}
        >
          {title || defaultTitle}
        </ModalHeader>
        <ModalBody>
          <p>{message}</p>
        </ModalBody>
        <ModalFooter>
          {type === "error" ? (
            <Button
              className="w-full"
              color="danger"
              variant="light"
              onPress={onClose}
            >
              Close
            </Button>
          ) : (
            <div className="flex w-full justify-end gap-2">
              <Button variant="flat" onPress={onClose}>
                Cancel
              </Button>
              <Button
                color="primary"
                onPress={() => {
                  if (onConfirm) onConfirm();
                  onClose();
                }}
              >
                Confirm
              </Button>
            </div>
          )}
        </ModalFooter>
      </ModalContent>
    </Modal>
  );
};

export default DialogModal;
