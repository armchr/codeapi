package com.example.petclinic.servlet;

import com.example.petclinic.model.OwnerDto;
import com.example.petclinic.model.PetDto;
import com.example.petclinic.model.VisitDto;
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
 * Servlet for handling Visit-related requests.
 */
@WebServlet(name = "VisitServlet", urlPatterns = {"/owners/*/pets/*/visits", "/owners/*/pets/*/visits/*"})
public class VisitServlet extends HttpServlet {
    private PetClinicApiClient apiClient;

    @Override
    public void init() throws ServletException {
        String apiBaseUrl = getServletContext().getInitParameter("apiBaseUrl");
        apiClient = apiBaseUrl != null ? new PetClinicApiClient(apiBaseUrl) : new PetClinicApiClient();
    }

    @Override
    protected void doGet(HttpServletRequest request, HttpServletResponse response)
            throws ServletException, IOException {
        String uri = request.getRequestURI();
        Long ownerId = extractOwnerId(uri);
        Long petId = extractPetId(uri);

        if (ownerId == null || petId == null) {
            response.sendError(HttpServletResponse.SC_BAD_REQUEST, "Invalid owner or pet ID");
            return;
        }

        try {
            OwnerDto owner = apiClient.getOwner(ownerId);
            PetDto pet = apiClient.getPet(ownerId, petId);
            request.setAttribute("owner", owner);
            request.setAttribute("pet", pet);

            String pathInfo = request.getPathInfo();
            if (pathInfo == null || pathInfo.endsWith("/visits") || pathInfo.endsWith("/visits/")) {
                // List visits for pet
                List<VisitDto> visits = apiClient.getVisitsByPet(ownerId, petId);
                request.setAttribute("visits", visits);
                request.getRequestDispatcher("/WEB-INF/jsp/visits/list.jsp").forward(request, response);

            } else if (pathInfo.contains("/visits/new")) {
                // Show create visit form
                request.setAttribute("visit", new VisitDto());
                request.setAttribute("action", "create");
                request.getRequestDispatcher("/WEB-INF/jsp/visits/form.jsp").forward(request, response);

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
        String uri = request.getRequestURI();
        Long ownerId = extractOwnerId(uri);
        Long petId = extractPetId(uri);

        if (ownerId == null || petId == null) {
            response.sendError(HttpServletResponse.SC_BAD_REQUEST, "Invalid owner or pet ID");
            return;
        }

        try {
            VisitDto visit = extractVisitFromRequest(request);
            visit.setPetId(petId);

            apiClient.createVisit(ownerId, petId, visit);
            response.sendRedirect(request.getContextPath() + "/owners/" + ownerId + "/pets/" + petId + "/visits");

        } catch (ApiException e) {
            request.setAttribute("error", e.getMessage());
            request.getRequestDispatcher("/WEB-INF/jsp/error.jsp").forward(request, response);
        }
    }

    private Long extractOwnerId(String uri) {
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

    private Long extractPetId(String uri) {
        String[] parts = uri.split("/");
        for (int i = 0; i < parts.length - 1; i++) {
            if (parts[i].equals("pets") && i + 1 < parts.length) {
                try {
                    return Long.parseLong(parts[i + 1]);
                } catch (NumberFormatException e) {
                    return null;
                }
            }
        }
        return null;
    }

    private VisitDto extractVisitFromRequest(HttpServletRequest request) {
        VisitDto visit = new VisitDto();
        visit.setDescription(request.getParameter("description"));

        String dateStr = request.getParameter("date");
        if (dateStr != null && !dateStr.isEmpty()) {
            try {
                visit.setDate(LocalDate.parse(dateStr));
            } catch (DateTimeParseException e) {
                visit.setDate(LocalDate.now());
            }
        } else {
            visit.setDate(LocalDate.now());
        }
        return visit;
    }
}
