package com.example.petclinic.servlet;

import com.example.petclinic.model.OwnerDto;
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
 * Servlet for handling Owner-related requests.
 */
@WebServlet(name = "OwnerServlet", urlPatterns = {"/owners", "/owners/*"})
public class OwnerServlet extends HttpServlet {
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
                // List all owners or search
                String lastName = request.getParameter("lastName");
                List<OwnerDto> owners = apiClient.searchOwners(lastName);
                request.setAttribute("owners", owners);
                request.setAttribute("searchTerm", lastName);
                request.getRequestDispatcher("/WEB-INF/jsp/owners/list.jsp").forward(request, response);

            } else if (pathInfo.equals("/new")) {
                // Show create form
                request.setAttribute("owner", new OwnerDto());
                request.setAttribute("action", "create");
                request.getRequestDispatcher("/WEB-INF/jsp/owners/form.jsp").forward(request, response);

            } else if (pathInfo.matches("/\\d+")) {
                // View single owner
                Long id = Long.parseLong(pathInfo.substring(1));
                OwnerDto owner = apiClient.getOwner(id);
                request.setAttribute("owner", owner);
                request.getRequestDispatcher("/WEB-INF/jsp/owners/view.jsp").forward(request, response);

            } else if (pathInfo.matches("/\\d+/edit")) {
                // Show edit form
                Long id = Long.parseLong(pathInfo.split("/")[1]);
                OwnerDto owner = apiClient.getOwner(id);
                request.setAttribute("owner", owner);
                request.setAttribute("action", "edit");
                request.getRequestDispatcher("/WEB-INF/jsp/owners/form.jsp").forward(request, response);

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

        try {
            OwnerDto owner = extractOwnerFromRequest(request);

            if (pathInfo == null || pathInfo.equals("/")) {
                // Create new owner
                OwnerDto created = apiClient.createOwner(owner);
                response.sendRedirect(request.getContextPath() + "/owners/" + created.getId());

            } else if (pathInfo.matches("/\\d+")) {
                // Update existing owner
                Long id = Long.parseLong(pathInfo.substring(1));
                owner.setId(id);
                apiClient.updateOwner(id, owner);
                response.sendRedirect(request.getContextPath() + "/owners/" + id);

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

        try {
            if (pathInfo != null && pathInfo.matches("/\\d+")) {
                Long id = Long.parseLong(pathInfo.substring(1));
                apiClient.deleteOwner(id);
                response.setStatus(HttpServletResponse.SC_NO_CONTENT);
            } else {
                response.sendError(HttpServletResponse.SC_NOT_FOUND);
            }
        } catch (ApiException e) {
            response.sendError(HttpServletResponse.SC_INTERNAL_SERVER_ERROR, e.getMessage());
        }
    }

    private OwnerDto extractOwnerFromRequest(HttpServletRequest request) {
        OwnerDto owner = new OwnerDto();
        owner.setFirstName(request.getParameter("firstName"));
        owner.setLastName(request.getParameter("lastName"));
        owner.setAddress(request.getParameter("address"));
        owner.setCity(request.getParameter("city"));
        owner.setTelephone(request.getParameter("telephone"));
        owner.setEmail(request.getParameter("email"));
        return owner;
    }
}
