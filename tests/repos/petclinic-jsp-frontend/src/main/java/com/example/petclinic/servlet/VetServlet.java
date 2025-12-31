package com.example.petclinic.servlet;

import com.example.petclinic.model.VetDto;
import com.example.petclinic.service.ApiException;
import com.example.petclinic.service.PetClinicApiClient;
import jakarta.servlet.ServletException;
import jakarta.servlet.annotation.WebServlet;
import jakarta.servlet.http.HttpServlet;
import jakarta.servlet.http.HttpServletRequest;
import jakarta.servlet.http.HttpServletResponse;

import java.io.IOException;
import java.util.List;

/**
 * Servlet for handling Vet-related requests.
 */
@WebServlet(name = "VetServlet", urlPatterns = {"/vets", "/vets/*"})
public class VetServlet extends HttpServlet {
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

        try {
            if (pathInfo == null || pathInfo.equals("/")) {
                // List all vets
                List<VetDto> vets = apiClient.getAllVets();
                request.setAttribute("vets", vets);
                request.getRequestDispatcher("/WEB-INF/jsp/vets/list.jsp").forward(request, response);

            } else if (pathInfo.matches("/\\d+")) {
                // View single vet
                Long id = Long.parseLong(pathInfo.substring(1));
                VetDto vet = apiClient.getVet(id);
                request.setAttribute("vet", vet);
                request.getRequestDispatcher("/WEB-INF/jsp/vets/view.jsp").forward(request, response);

            } else {
                response.sendError(HttpServletResponse.SC_NOT_FOUND);
            }
        } catch (ApiException e) {
            request.setAttribute("error", e.getMessage());
            request.getRequestDispatcher("/WEB-INF/jsp/error.jsp").forward(request, response);
        }
    }
}
