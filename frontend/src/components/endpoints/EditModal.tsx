import { useState, useEffect } from "react";
import { Button } from "@nextui-org/button";
import { Input } from "@nextui-org/input";
import { RadioGroup, Radio } from "@nextui-org/radio";
import {
  Modal,
  ModalContent,
  ModalHeader,
  ModalBody,
  ModalFooter,
} from "@nextui-org/modal";
import { addToast } from "@nextui-org/toast";
import { Form } from "@nextui-org/form";

import { endpointApi, EnhancedAxiosError } from "@/api";
import { EndpointUpdate } from "@/types";
import LoadingSpinner from "@/components/LoadingSpinner";

interface EndpointEditModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSuccess: () => void;
  endpointId: number | undefined;
  endpointName: string;
  endpointUrl: string;
  endpointType?: string;
  apiKey?: string;
  onDelete?: (id: number) => void;
}

const EndpointEditModal = ({
  isOpen,
  onClose,
  onSuccess,
  endpointId,
  endpointName,
  endpointUrl,
  endpointType,
  apiKey,
}: EndpointEditModalProps) => {
  // Form state
  const [formData, setFormData] = useState<EndpointUpdate>({
    name: "",
    endpoint_type: undefined,
    api_key: undefined,
  });
  const [isSubmitting, setIsSubmitting] = useState(false);

  // Update form data when props change
  useEffect(() => {
    if (isOpen) {
      setFormData({
        name: endpointName,
        endpoint_type: endpointType,
        api_key: apiKey || "",
      });
    }
  }, [endpointName, endpointType, apiKey, isOpen]);

  // Reset state on close
  const handleClose = () => {
    if (!isSubmitting) {
      onClose();
    }
  };

  // Handle form input change
  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const { name, value } = e.target;

    setFormData((prev) => ({
      ...prev,
      [name]: value,
    }));
  };

  // Handle form submit
  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!endpointId) return;

    setIsSubmitting(true);

    try {
      // Form validation
      if (!formData.name) {
        throw new Error("Name cannot be empty");
      }

      // Submit update request
      await endpointApi.updateEndpoint(endpointId, formData);

      // Success, close modal
      setIsSubmitting(false);
      onSuccess();
      handleClose();
    } catch (err) {
      addToast({
        title: "Failed to update endpoint",
        description: (err as EnhancedAxiosError).detail || "Failed to update endpoint",
        color: "danger",
      });
      setIsSubmitting(false);
    }
  };

  return (
    <Modal isOpen={isOpen} placement="center" onClose={handleClose}>
      <ModalContent>
        <Form className="w-full" onSubmit={handleSubmit}>
          <ModalHeader>Edit Endpoint</ModalHeader>
          <ModalBody className="w-full">
            <div className="space-y-4">
              <div className="mb-4">
                <Input
                  disabled
                  className="w-full"
                  description="Endpoint URL cannot be modified"
                  id="url"
                  label="Endpoint URL"
                  value={endpointUrl}
                />
              </div>

              <div className="mb-6">
                <Input
                  className="w-full"
                  description="Endpoint Name"
                  id="name"
                  label="Endpoint Name"
                  name="name"
                  placeholder="Endpoint Name"
                  value={formData.name}
                  onChange={handleInputChange}
                />
              </div>

              <div className="mb-4">
                <RadioGroup
                  label="Endpoint Type"
                  description="Select the type of API endpoint"
                  value={formData.endpoint_type || "ollama"}
                  onValueChange={(value) =>
                    setFormData((prev) => ({
                      ...prev,
                      endpoint_type: value,
                    }))
                  }
                >
                  <Radio value="ollama">
                    Ollama (Native API)
                  </Radio>
                  <Radio value="openai">
                    OpenAI Compatible (e.g., /v1/models)
                  </Radio>
                </RadioGroup>
              </div>

              {formData.endpoint_type === "openai" && (
                <div className="mb-4">
                  <Input
                    className="w-full"
                    description="API key for authentication (e.g., sk-...)"
                    id="api_key"
                    label="API Key"
                    name="api_key"
                    placeholder="sk-..."
                    type="password"
                    value={formData.api_key || ""}
                    onChange={handleInputChange}
                  />
                </div>
              )}
            </div>
          </ModalBody>
          <ModalFooter className="w-full">
            <Button
              color="primary"
              disabled={isSubmitting}
              isLoading={isSubmitting}
              type="submit"
            >
              {isSubmitting ? (
                <>
                  <LoadingSpinner size="small" />
                  <span className="ml-2">Saving...</span>
                </>
              ) : (
                "Save Changes"
              )}
            </Button>
          </ModalFooter>
        </Form>
      </ModalContent>
    </Modal>
  );
};

export default EndpointEditModal;
