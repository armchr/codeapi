package com.example.petclinic.servlet;

import com.example.petclinic.model.OwnerDto;
import com.example.petclinic.model.PetDto;
import com.example.petclinic.model.PetTypeDto;
import com.example.petclinic.service.ApiException;
import com.example.petclinic.service.PetClinicApiClient;
import jakarta.servlet.ServletException;
import jakarta.servlet.annotation.WebServlet;
import jakarta.servlet.http.HttpServlet;
import jakarta.servlet.http.HttpServletRequest;
import jakarta.servlet.http.HttpServletResponse;

import java.io.IOException;
import java.time.LocalDate;
import java.time.format.DateTimeParseException;
import java.util.List;

/**
 * Servlet for handling Pet-related requests.
 */
@WebServlet(name = "PetServlet", urlPatterns = {"/owners/*/pets", "/owners/*/pets/*"})
public class PetServlet extends HttpServlet {
    private PetClinicApiClient apiClient;

    @Override
    public void init() throws ServletException {
        String apiBaseUrl = getServletContext().getInitParameter("apiBaseUrl");
        apiClient = apiBaseUrl != null ? new PetClinicApiClient(apiBaseUrl) : new PetClinicApiClient();
    }

    @Override
    protected void doGet(HttpServletRequest request, HttpServletResponse response)
            throws ServletException, IOException {
        String pathInfo = request.getPathInfo();
        Long ownerId = extractOwnerId(request.getRequestURI());

        if (ownerId == null) {
            response.sendError(HttpServletResponse.SC_BAD_REQUEST, "Invalid owner ID");
            return;
        }

        try {
            OwnerDto owner = apiClient.getOwner(ownerId);
            request.setAttribute("owner", owner);

            if (pathInfo == null || pathInfo.endsWith("/pets") || pathInfo.endsWith("/pets/")) {
                // List pets for owner (already in owner.getPets())
                request.getRequestDispatcher("/WEB-INF/jsp/pets/list.jsp").forward(request, response);

            } else if (pathInfo.matches(".*/pets/new")) {
                // Show create pet form
                List<PetTypeDto> petTypes = apiClient.getPetTypes();
                request.setAttribute("pet", new PetDto());
                request.setAttribute("petTypes", petTypes);
                request.setAttribute("action", "create");
                request.getRequestDispatcher("/WEB-INF/jsp/pets/form.jsp").forward(request, response);

            } else if (pathInfo.matches(".*/pets/\\d+$")) {
                // View single pet
                Long petId = extractPetId(pathInfo);
                PetDto pet = apiClient.getPet(ownerId, petId);
                request.setAttribute("pet", pet);
                request.getRequestDispatcher("/WEB-INF/jsp/pets/view.jsp").forward(request, response);

            } else if (pathInfo.matches(".*/pets/\\d+/edit")) {
                // Show edit pet form
                Long petId = extractPetIdFromEdit(pathInfo);
                PetDto pet = apiClient.getPet(ownerId, petId);
                List<PetTypeDto> petTypes = apiClient.getPetTypes();
                request.setAttribute("pet", pet);
                request.setAttribute("petTypes", petTypes);
                request.setAttribute("action", "edit");
                request.getRequestDispatcher("/WEB-INF/jsp/pets/form.jsp").forward(request, response);

            } else {
                response.sendError(HttpServletResponse.SC_NOT_FOUND);
            }
        } catch (ApiException e) {
            request.setAttribute("error", e.getMessage());
            request.getRequestDispatcher("/WEB-INF/jsp/error.jsp").forward(request, response);
        }
    }

    @Override
    protected void doPost(HttpServletRequest request, HttpServletResponse response)
            throws ServletException, IOException {
        String pathInfo = request.getPathInfo();
        Long ownerId = extractOwnerId(request.getRequestURI());

        if (ownerId == null) {
            response.sendError(HttpServletResponse.SC_BAD_REQUEST, "Invalid owner ID");
            return;
        }

        try {
            PetDto pet = extractPetFromRequest(request);
            pet.setOwnerId(ownerId);

            if (pathInfo == null || pathInfo.endsWith("/pets") || pathInfo.endsWith("/pets/")) {
                // Create new pet
                PetDto created = apiClient.createPet(ownerId, pet);
                response.sendRedirect(request.getContextPath() + "/owners/" + ownerId);

            } else if (pathInfo.matches(".*/pets/\\d+$")) {
                // Update existing pet
                Long petId = extractPetId(pathInfo);
                pet.setId(petId);
                apiClient.updatePet(ownerId, petId, pet);
                response.sendRedirect(request.getContextPath() + "/owners/" + ownerId);

            } else {
                response.sendError(HttpServletResponse.SC_NOT_FOUND);
            }
        } catch (ApiException e) {
            request.setAttribute("error", e.getMessage());
            request.getRequestDispatcher("/WEB-INF/jsp/error.jsp").forward(request, response);
        }
    }

    @Override
    protected void doDelete(HttpServletRequest request, HttpServletResponse response)
            throws ServletException, IOException {
        String pathInfo = request.getPathInfo();
        Long ownerId = extractOwnerId(request.getRequestURI());

        if (ownerId == null) {
            response.sendError(HttpServletResponse.SC_BAD_REQUEST, "Invalid owner ID");
            return;
        }

        try {
            if (pathInfo != null && pathInfo.matches(".*/pets/\\d+$")) {
                Long petId = extractPetId(pathInfo);
                apiClient.deletePet(ownerId, petId);
                response.setStatus(HttpServletResponse.SC_NO_CONTENT);
            } else {
                response.sendError(HttpServletResponse.SC_NOT_FOUND);
            }
        } catch (ApiException e) {
            response.sendError(HttpServletResponse.SC_INTERNAL_SERVER_ERROR, e.getMessage());
        }
    }

    private Long extractOwnerId(String uri) {
        // URI pattern: /context/owners/{ownerId}/pets...
        String[] parts = uri.split("/");
        for (int i = 0; i < parts.length - 1; i++) {
            if (parts[i].equals("owners") && i + 1 < parts.length) {
                try {
                    return Long.parseLong(parts[i + 1]);
                } catch (NumberFormatException e) {
                    return null;
                }
            }
        }
        return null;
    }

    private Long extractPetId(String pathInfo) {
        // pathInfo pattern: .../pets/{petId}
        String[] parts = pathInfo.split("/");
        return Long.parseLong(parts[parts.length - 1]);
    }

    private Long extractPetIdFromEdit(String pathInfo) {
        // pathInfo pattern: .../pets/{petId}/edit
        String[] parts = pathInfo.split("/");
        return Long.parseLong(parts[parts.length - 2]);
    }

    private PetDto extractPetFromRequest(HttpServletRequest request) {
        PetDto pet = new PetDto();
        pet.setName(request.getParameter("name"));
        pet.setTypeName(request.getParameter("typeName"));

        String birthDateStr = request.getParameter("birthDate");
        if (birthDateStr != null && !birthDateStr.isEmpty()) {
            try {
                pet.setBirthDate(LocalDate.parse(birthDateStr));
            } catch (DateTimeParseException e) {
                // Leave birthDate as null
            }
        }
        return pet;
    }
}
